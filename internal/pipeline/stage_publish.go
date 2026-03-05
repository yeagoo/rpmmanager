package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// StagePublish atomically swaps staging to production with backup.
func StagePublish(ctx context.Context, repoRoot, productName, stagingDir string, log *LogWriter) error {
	productionDir := filepath.Join(repoRoot, productName)
	rollbackDir := filepath.Join(repoRoot, ".rollback")

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

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
