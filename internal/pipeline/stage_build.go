package pipeline

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/distromap"
	"github.com/ivmm/rpmmanager/internal/models"
)

// StageBuild downloads upstream binary, builds RPMs with nfpm, creates repo metadata, symlinks, and .repo templates.
func StageBuild(ctx context.Context, cfg *config.Config, product *models.Product, version string, stagingDir string, log *LogWriter) (rpmCount int, symlinkCount int, err error) {
	// Resolve product lines
	var customLines []distromap.ProductLine
	productLines, resolveErr := distromap.ResolveAll(product.TargetDistros, customLines)
	if resolveErr != nil {
		return 0, 0, fmt.Errorf("resolve product lines: %w", resolveErr)
	}

	log.WriteLog("Resolved %d product lines for %d distros", len(productLines), len(product.TargetDistros))

	// Download binaries for each architecture
	binaries := make(map[string]string) // arch -> binary path
	for _, arch := range product.Architectures {
		binPath, dlErr := downloadBinary(ctx, cfg, product, version, arch, log)
		if dlErr != nil {
			return 0, 0, fmt.Errorf("download binary for %s: %w", arch, dlErr)
		}
		binaries[arch] = binPath
	}

	// Build RPMs: for each (product_line, architecture)
	for _, pl := range productLines {
		for _, arch := range product.Architectures {
			binPath := binaries[arch]

			rpmBuilt, buildErr := buildRPMForArch(ctx, cfg, product, version, arch, &pl, binPath, stagingDir, log)
			if buildErr != nil {
				return rpmCount, 0, buildErr
			}
			if rpmBuilt {
				rpmCount++
			}
		}

		// Generate repodata for each arch
		for _, arch := range product.Architectures {
			repoDir := filepath.Join(stagingDir, pl.Path, arch)
			if err := RunCreaterepo(ctx, cfg.Tools.CreaterepoPath, repoDir, log); err != nil {
				return rpmCount, 0, fmt.Errorf("createrepo %s/%s: %w", pl.ID, arch, err)
			}
		}
	}

	// Create symlinks
	symlinkCount, err = CreateSymlinks(stagingDir, product.TargetDistros, customLines, log)
	if err != nil {
		return rpmCount, symlinkCount, fmt.Errorf("create symlinks: %w", err)
	}

	// Generate .repo templates
	baseURL := product.BaseURL
	if baseURL == "" {
		baseURL = cfg.Server.BaseURL
	}
	if err := GenerateRepoTemplates(stagingDir, product.Name, baseURL, product.TargetDistros, customLines, log); err != nil {
		return rpmCount, symlinkCount, fmt.Errorf("generate repo templates: %w", err)
	}

	return rpmCount, symlinkCount, nil
}

// buildRPMForArch builds a single RPM for a product line and architecture.
func buildRPMForArch(ctx context.Context, cfg *config.Config, product *models.Product, version, arch string, pl *distromap.ProductLine, binPath, stagingDir string, log *LogWriter) (bool, error) {
	targetDir := filepath.Join(stagingDir, pl.Path, arch, "Packages")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return false, fmt.Errorf("create target dir: %w", err)
	}

	tempDir := filepath.Join(cfg.Storage.TempDir, fmt.Sprintf("nfpm-%s-%s-%s", product.Name, pl.ID, arch))
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	configPath, genErr := GenerateNfpmYAML(product, version, arch, pl, binPath, tempDir)
	if genErr != nil {
		return false, fmt.Errorf("generate nfpm config for %s/%s: %w", pl.ID, arch, genErr)
	}

	log.WriteLog("Building RPM: %s/%s/%s", product.Name, pl.ID, arch)
	cmd := exec.CommandContext(ctx, cfg.Tools.NfpmPath,
		"package",
		"--config", configPath,
		"--packager", "rpm",
		"--target", targetDir,
	)
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("nfpm build %s/%s: %w", pl.ID, arch, err)
	}
	return true, nil
}

// httpClient is a shared HTTP client with a reasonable timeout for binary downloads.
var httpClient = &http.Client{
	Timeout: 10 * time.Minute,
}

// downloadBinary downloads the upstream binary for a given product, version, and arch.
func downloadBinary(ctx context.Context, cfg *config.Config, product *models.Product, version, arch string, log *LogWriter) (string, error) {
	// Map arch for download
	downloadArch := arch
	if arch == "x86_64" {
		downloadArch = "amd64"
	} else if arch == "aarch64" {
		downloadArch = "arm64"
	}

	var url string
	if product.SourceType == "github" {
		// Build asset filename from pattern or use default
		assetPattern := product.SourceGithubAssetPattern
		if assetPattern == "" {
			assetPattern = "{repo}_{version}_linux_{arch}"
		}
		assetName := assetPattern
		assetName = strings.ReplaceAll(assetName, "{repo}", product.SourceGithubRepo)
		assetName = strings.ReplaceAll(assetName, "{owner}", product.SourceGithubOwner)
		assetName = strings.ReplaceAll(assetName, "{version}", version)
		assetName = strings.ReplaceAll(assetName, "{arch}", downloadArch)

		url = fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s",
			product.SourceGithubOwner, product.SourceGithubRepo, version, assetName)
	} else {
		// Custom URL template
		url = product.SourceURLTemplate
		url = strings.ReplaceAll(url, "{version}", version)
		url = strings.ReplaceAll(url, "{arch}", downloadArch)
	}

	log.WriteLog("Downloading: %s", url)

	// Create temp file for download
	binDir := filepath.Join(cfg.Storage.TempDir, "binaries", product.Name)
	os.MkdirAll(binDir, 0755)
	binPath := filepath.Join(binDir, fmt.Sprintf("%s-%s-%s", product.Name, version, arch))

	// Download
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	// Check if the URL indicates a tar.gz archive
	if strings.HasSuffix(url, ".tar.gz") || strings.HasSuffix(url, ".tgz") {
		// Download and extract binary from tar.gz
		return extractBinaryFromTarGz(resp.Body, binPath, product.Name, log)
	}

	// Direct binary download
	f, err := os.Create(binPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return "", fmt.Errorf("write binary: %w", err)
	}

	os.Chmod(binPath, 0755)
	log.WriteLog("Downloaded %s (%d bytes)", filepath.Base(binPath), written)

	return binPath, nil
}

// extractBinaryFromTarGz extracts a binary from a tar.gz archive.
// It looks for an executable file matching the product name.
func extractBinaryFromTarGz(reader io.Reader, binPath, productName string, log *LogWriter) (string, error) {
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return "", fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read tar: %w", err)
		}

		// Skip directories and non-regular files
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		// Match the binary: look for a file whose base name equals the product name
		baseName := filepath.Base(hdr.Name)
		if baseName == productName {
			written, copyErr := extractFileFromTar(tr, binPath)
			if copyErr != nil {
				os.Remove(binPath)
				return "", fmt.Errorf("extract binary: %w", copyErr)
			}

			os.Chmod(binPath, 0755)
			log.WriteLog("Extracted %s from archive (%d bytes)", baseName, written)
			return binPath, nil
		}
	}

	return "", fmt.Errorf("binary %q not found in tar.gz archive", productName)
}

// extractFileFromTar writes a tar entry to disk, cleaning up on failure.
func extractFileFromTar(tr *tar.Reader, destPath string) (int64, error) {
	f, err := os.Create(destPath)
	if err != nil {
		return 0, err
	}

	written, err := io.Copy(f, tr)
	if closeErr := f.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	return written, err
}
