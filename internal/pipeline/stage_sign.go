package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/ivmm/rpmmanager/internal/config"
)

// StageSign signs all RPM packages and repomd.xml files.
func StageSign(ctx context.Context, cfg *config.Config, gpgKeyID string, gpgHomeDir string, stagingDir string, log *LogWriter) error {
	if gpgKeyID == "" {
		log.WriteLog("No GPG key configured, skipping signing")
		return nil
	}

	// Find all RPM files
	rpmFiles, err := filepath.Glob(filepath.Join(stagingDir, "*", "*", "Packages", "*.rpm"))
	if err != nil {
		return fmt.Errorf("glob RPM files: %w", err)
	}

	// Sign each RPM
	for _, rpmFile := range rpmFiles {
		log.WriteLog("Signing RPM: %s", filepath.Base(rpmFile))
		cmd := exec.CommandContext(ctx, cfg.Tools.RPMPath,
			"--addsign",
			"--define", fmt.Sprintf("%%_gpg_name %s", gpgKeyID),
			rpmFile,
		)
		if gpgHomeDir != "" {
			cmd.Env = append(cmd.Environ(), fmt.Sprintf("GNUPGHOME=%s", gpgHomeDir))
		}
		cmd.Stdout = log
		cmd.Stderr = log
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("rpm --addsign %s: %w", filepath.Base(rpmFile), err)
		}
	}

	// Find all repomd.xml files and create detached signatures
	repomdFiles, err := filepath.Glob(filepath.Join(stagingDir, "*", "*", "repodata", "repomd.xml"))
	if err != nil {
		return fmt.Errorf("glob repomd.xml files: %w", err)
	}

	for _, repomdFile := range repomdFiles {
		ascFile := repomdFile + ".asc"
		log.WriteLog("Signing repomd: %s", repomdFile)
		cmd := exec.CommandContext(ctx, cfg.Tools.GPGPath,
			"--batch", "--yes",
			"--detach-sign", "--armor",
			"--local-user", gpgKeyID,
			"--output", ascFile,
			repomdFile,
		)
		if gpgHomeDir != "" {
			cmd.Env = append(cmd.Environ(), fmt.Sprintf("GNUPGHOME=%s", gpgHomeDir))
		}
		cmd.Stdout = log
		cmd.Stderr = log
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gpg sign repomd.xml: %w", err)
		}
	}

	// Export GPG public key
	gpgKeyFile := filepath.Join(stagingDir, "gpg.key")
	log.WriteLog("Exporting GPG public key to gpg.key")
	cmd := exec.CommandContext(ctx, cfg.Tools.GPGPath,
		"--batch",
		"--export", "--armor",
		gpgKeyID,
	)
	if gpgHomeDir != "" {
		cmd.Env = append(cmd.Environ(), fmt.Sprintf("GNUPGHOME=%s", gpgHomeDir))
	}
	var keyBuf []byte
	keyBuf, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("gpg export public key: %w", err)
	}
	if err := writeFile(gpgKeyFile, keyBuf); err != nil {
		return fmt.Errorf("write gpg.key: %w", err)
	}

	log.WriteLog("Signed %d RPMs and %d repomd.xml files", len(rpmFiles), len(repomdFiles))
	return nil
}
