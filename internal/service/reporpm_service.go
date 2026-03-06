package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/distromap"
	"gopkg.in/yaml.v3"
)

// RepoRPMService generates repository RPM packages (like epel-release).
type RepoRPMService struct {
	cfg    *config.Config
	gpgSvc *GPGService
}

// RepoRPMRequest contains parameters for generating a repo RPM.
type RepoRPMRequest struct {
	ProductName string   `json:"product_name"`
	DisplayName string   `json:"display_name"`
	BaseURL     string   `json:"base_url"`
	GPGKeyID    int64    `json:"gpg_key_id"`
	Distros     []string `json:"distros"`
	Maintainer  string   `json:"maintainer"`
	Vendor      string   `json:"vendor"`
	Homepage    string   `json:"homepage"`
	Version     string   `json:"version"`
}

// RepoRPMResult is returned after successful generation.
type RepoRPMResult struct {
	FileName    string `json:"filename"`
	FilePath    string `json:"file_path"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
}

func NewRepoRPMService(cfg *config.Config, gpgSvc *GPGService) *RepoRPMService {
	return &RepoRPMService{cfg: cfg, gpgSvc: gpgSvc}
}

// Generate creates a noarch repo RPM containing .repo files and GPG public key.
func (s *RepoRPMService) Generate(req *RepoRPMRequest) (*RepoRPMResult, error) {
	if req.ProductName == "" {
		return nil, fmt.Errorf("product_name is required")
	}
	if req.GPGKeyID == 0 {
		return nil, fmt.Errorf("gpg_key_id is required")
	}
	if len(req.Distros) == 0 {
		return nil, fmt.Errorf("at least one distro is required")
	}
	if req.Version == "" {
		req.Version = "1.0"
	}
	if req.BaseURL == "" {
		req.BaseURL = s.cfg.Server.BaseURL
	}

	// Create temp working directory
	tempDir, err := os.MkdirTemp(s.cfg.Storage.TempDir, "repo-rpm-"+req.ProductName+"-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Export GPG public key
	gpgArmor, err := s.gpgSvc.Export(req.GPGKeyID)
	if err != nil {
		return nil, fmt.Errorf("export gpg key: %w", err)
	}

	gpgKeyFileName := fmt.Sprintf("RPM-GPG-KEY-%s", req.ProductName)
	gpgKeyPath := filepath.Join(tempDir, gpgKeyFileName)
	if err := os.WriteFile(gpgKeyPath, []byte(gpgArmor), 0644); err != nil {
		return nil, fmt.Errorf("write gpg key: %w", err)
	}

	// 2. Resolve distros to unique product lines
	productLines, err := distromap.ResolveAll(req.Distros, nil)
	if err != nil {
		return nil, fmt.Errorf("resolve distros: %w", err)
	}

	// 3. Generate .repo files for each product line
	var repoFiles []string
	for _, pl := range productLines {
		repoContent := s.generateRepoContent(req, &pl)
		repoFileName := fmt.Sprintf("%s-%s.repo", req.ProductName, pl.ID)
		repoPath := filepath.Join(tempDir, repoFileName)
		if err := os.WriteFile(repoPath, []byte(repoContent), 0644); err != nil {
			return nil, fmt.Errorf("write repo file %s: %w", repoFileName, err)
		}
		repoFiles = append(repoFiles, repoFileName)
	}

	// 4. Build nfpm YAML config
	nfpmConfig := s.buildNfpmConfig(req, repoFiles, gpgKeyFileName, tempDir)
	nfpmYAML, err := yaml.Marshal(nfpmConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal nfpm config: %w", err)
	}

	nfpmConfigPath := filepath.Join(tempDir, "nfpm.yaml")
	if err := os.WriteFile(nfpmConfigPath, nfpmYAML, 0644); err != nil {
		return nil, fmt.Errorf("write nfpm config: %w", err)
	}

	// 5. Create output directory
	outputDir := filepath.Join(s.cfg.Storage.RepoRoot, req.ProductName, "repo-rpm")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	// 6. Run nfpm to build the RPM
	cmd := exec.Command(s.cfg.Tools.NfpmPath,
		"package",
		"--config", nfpmConfigPath,
		"--packager", "rpm",
		"--target", outputDir,
	)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("nfpm build failed: %w\nOutput: %s", err, string(output))
	}

	// 7. Find the generated RPM file
	rpmFileName := fmt.Sprintf("%s-repo-%s-1.noarch.rpm", req.ProductName, req.Version)
	rpmFilePath := filepath.Join(outputDir, rpmFileName)

	// nfpm might name it slightly differently, so scan for the file
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, fmt.Errorf("read output dir: %w", err)
	}

	// Find the newest RPM file
	var latestFile string
	var latestMod int64
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".rpm") && strings.Contains(entry.Name(), req.ProductName+"-repo") {
			info, _ := entry.Info()
			if info != nil && info.ModTime().UnixNano() > latestMod {
				latestMod = info.ModTime().UnixNano()
				latestFile = entry.Name()
			}
		}
	}
	if latestFile != "" {
		rpmFileName = latestFile
		rpmFilePath = filepath.Join(outputDir, rpmFileName)
	}

	info, err := os.Stat(rpmFilePath)
	if err != nil {
		return nil, fmt.Errorf("generated RPM not found at %s: %w", rpmFilePath, err)
	}

	return &RepoRPMResult{
		FileName:    rpmFileName,
		FilePath:    rpmFilePath,
		Size:        info.Size(),
		DownloadURL: fmt.Sprintf("/api/products/%s/repo-rpm/download", req.ProductName),
	}, nil
}

// GetLatest returns the latest repo RPM for a product.
func (s *RepoRPMService) GetLatest(productName string) (*RepoRPMResult, error) {
	outputDir := filepath.Join(s.cfg.Storage.RepoRoot, productName, "repo-rpm")

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read repo-rpm dir: %w", err)
	}

	var latestFile string
	var latestMod int64
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".rpm") {
			info, _ := entry.Info()
			if info != nil && info.ModTime().UnixNano() > latestMod {
				latestMod = info.ModTime().UnixNano()
				latestFile = entry.Name()
			}
		}
	}

	if latestFile == "" {
		return nil, nil
	}

	filePath := filepath.Join(outputDir, latestFile)
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return &RepoRPMResult{
		FileName: latestFile,
		FilePath: filePath,
		Size:     info.Size(),
	}, nil
}

// GetFilePath returns the full path for a specific repo RPM file, with path traversal protection.
func (s *RepoRPMService) GetFilePath(productName, fileName string) (string, error) {
	outputDir := filepath.Join(s.cfg.Storage.RepoRoot, productName, "repo-rpm")
	absDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(absDir, filepath.Base(fileName))
	if !strings.HasPrefix(filePath, absDir+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid file path")
	}
	if !strings.HasSuffix(filePath, ".rpm") {
		return "", fmt.Errorf("invalid file type")
	}

	if _, err := os.Stat(filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

func (s *RepoRPMService) generateRepoContent(req *RepoRPMRequest, pl *distromap.ProductLine) string {
	repoID := fmt.Sprintf("%s-%s", req.ProductName, pl.ID)
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.ProductName
	}
	repoName := fmt.Sprintf("%s for %s", displayName, pl.ID)
	baseURL := fmt.Sprintf("%s/%s/%s/$basearch/", req.BaseURL, req.ProductName, pl.Path)

	return fmt.Sprintf(`[%s]
name=%s
baseurl=%s
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-%s
`, repoID, repoName, baseURL, req.ProductName)
}

func (s *RepoRPMService) buildNfpmConfig(req *RepoRPMRequest, repoFiles []string, gpgKeyFileName, tempDir string) map[string]interface{} {
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.ProductName
	}

	// Build contents list
	var contents []map[string]interface{}

	// Add GPG key
	contents = append(contents, map[string]interface{}{
		"src": filepath.Join(tempDir, gpgKeyFileName),
		"dst": fmt.Sprintf("/etc/pki/rpm-gpg/%s", gpgKeyFileName),
	})

	// Add .repo files
	for _, repoFile := range repoFiles {
		contents = append(contents, map[string]interface{}{
			"src": filepath.Join(tempDir, repoFile),
			"dst": fmt.Sprintf("/etc/yum.repos.d/%s", repoFile),
		})
	}

	config := map[string]interface{}{
		"name":        req.ProductName + "-repo",
		"arch":        "noarch",
		"version":     req.Version,
		"release":     "1",
		"maintainer":  req.Maintainer,
		"description": fmt.Sprintf("YUM/DNF repository configuration for %s", displayName),
		"vendor":      req.Vendor,
		"homepage":    req.Homepage,
		"license":     "MIT",
		"contents":    contents,
		"rpm": map[string]interface{}{
			"group": "System Environment/Base",
		},
	}

	return config
}
