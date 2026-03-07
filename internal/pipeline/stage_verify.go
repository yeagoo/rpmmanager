package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ivmm/rpmmanager/internal/config"
)

// StageVerify runs rpmlint, repodata validation, and signature verification.
func StageVerify(ctx context.Context, cfg *config.Config, productDir string, gpgKeyID string, log *LogWriter) error {
	// Find all RPM files
	rpmFiles, err := filepath.Glob(filepath.Join(productDir, "*", "*", "Packages", "*.rpm"))
	if err != nil {
		return fmt.Errorf("glob RPM files: %w", err)
	}

	log.WriteLog("Verifying %d RPM packages", len(rpmFiles))

	// rpmlint (optional, non-fatal)
	if cfg.Tools.RPMLintPath != "" {
		for _, rpmFile := range rpmFiles {
			cmd := exec.CommandContext(ctx, cfg.Tools.RPMLintPath, rpmFile)
			cmd.Stdout = log
			cmd.Stderr = log
			if err := cmd.Run(); err != nil {
				log.WriteLog("Warning: rpmlint issues in %s (non-fatal)", filepath.Base(rpmFile))
			}
		}
	}

	// Verify repomd.xml exists
	repomdFiles, err := filepath.Glob(filepath.Join(productDir, "*", "*", "repodata", "repomd.xml"))
	if err != nil {
		return fmt.Errorf("glob repomd.xml: %w", err)
	}

	for _, repomdFile := range repomdFiles {
		content, err := os.ReadFile(repomdFile)
		if err != nil {
			return fmt.Errorf("read repomd.xml %s: %w", repomdFile, err)
		}
		if len(content) == 0 {
			return fmt.Errorf("repomd.xml is empty: %s", repomdFile)
		}
		log.WriteLog("Verified: %s (%d bytes)", repomdFile, len(content))
	}

	// Verify RPM signatures (if GPG key is configured)
	if gpgKeyID != "" {
		// Import the GPG public key into RPM keyring so rpm -K can verify
		gpgKeyFile := filepath.Join(productDir, "gpg.key")
		if _, err := os.Stat(gpgKeyFile); err == nil {
			cmd := exec.CommandContext(ctx, cfg.Tools.RPMPath, "--import", gpgKeyFile)
			cmd.Stdout = log
			cmd.Stderr = log
			if err := cmd.Run(); err != nil {
				log.WriteLog("Warning: could not import GPG key into RPM keyring: %v (signature check may fail)", err)
			}
		}

		for _, rpmFile := range rpmFiles {
			cmd := exec.CommandContext(ctx, cfg.Tools.RPMPath, "-K", rpmFile)
			cmd.Stdout = log
			cmd.Stderr = log
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("RPM signature verification failed for %s: %w", filepath.Base(rpmFile), err)
			}
		}
	}

	// Verify symlinks
	entries, err := os.ReadDir(productDir)
	if err != nil {
		return fmt.Errorf("read product dir: %w", err)
	}
	for _, e := range entries {
		fullPath := filepath.Join(productDir, e.Name())
		info, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(fullPath)
			if err != nil {
				return fmt.Errorf("read symlink %s: %w", e.Name(), err)
			}
			resolvedTarget := filepath.Join(filepath.Dir(fullPath), target)
			if _, err := os.Stat(resolvedTarget); err != nil {
				return fmt.Errorf("broken symlink %s -> %s", e.Name(), target)
			}
			log.WriteLog("Verified symlink: %s -> %s", e.Name(), target)
		}
	}

	log.WriteLog("Verification complete: %d RPMs, %d repos validated", len(rpmFiles), len(repomdFiles))
	return nil
}
