package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// preserveDirs lists subdirectories that should survive across rebuilds.
// These are copied from the old production dir into staging before the atomic swap.
var preserveDirs = []string{"repo-rpm"}

// StagePublish atomically swaps staging to production with backup.
func StagePublish(ctx context.Context, repoRoot, productName, stagingDir string, log *LogWriter) error {
	productionDir := filepath.Join(repoRoot, productName)
	rollbackDir := filepath.Join(repoRoot, ".rollback")

	// Preserve directories that are not part of the build output
	if _, err := os.Stat(productionDir); err == nil {
		for _, dir := range preserveDirs {
			src := filepath.Join(productionDir, dir)
			dst := filepath.Join(stagingDir, dir)
			if info, statErr := os.Stat(src); statErr == nil && info.IsDir() {
				log.WriteLog("Preserving %s/ from previous build", dir)
				if copyErr := copyDir(src, dst); copyErr != nil {
					log.WriteLog("Warning: failed to preserve %s: %v", dir, copyErr)
				}
			}
		}
	}

	// Backup existing production if it exists
	if _, err := os.Stat(productionDir); err == nil {
		timestamp := time.Now().Format("20060102-150405")
		backupDir := filepath.Join(rollbackDir, timestamp, productName)
		if err := os.MkdirAll(filepath.Dir(backupDir), 0755); err != nil {
			return fmt.Errorf("create backup dir: %w", err)
		}

		log.WriteLog("Backing up %s to %s", productionDir, backupDir)
		if err := os.Rename(productionDir, backupDir); err != nil {
			return fmt.Errorf("backup production dir: %w", err)
		}
	}

	// Atomic swap: staging -> production
	log.WriteLog("Publishing: %s -> %s", stagingDir, productionDir)
	if err := os.MkdirAll(filepath.Dir(productionDir), 0755); err != nil {
		return fmt.Errorf("create production parent: %w", err)
	}
	if err := os.Rename(stagingDir, productionDir); err != nil {
		return fmt.Errorf("atomic publish: %w", err)
	}

	// Clean old backups (keep latest 3)
	cleanOldBackups(rollbackDir, 3, log)

	log.WriteLog("Published successfully")
	return nil
}

func cleanOldBackups(rollbackDir string, keep int, log *LogWriter) {
	entries, err := os.ReadDir(rollbackDir)
	if err != nil {
		return
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}

	sort.Strings(dirs)

	if len(dirs) <= keep {
		return
	}

	for _, d := range dirs[:len(dirs)-keep] {
		path := filepath.Join(rollbackDir, d)
		log.WriteLog("Removing old backup: %s", d)
		os.RemoveAll(path)
	}
}

// copyDir recursively copies a directory tree.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
