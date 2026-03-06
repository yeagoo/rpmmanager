package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// preserveDirs lists subdirectories that should survive across rebuilds.
// These are copied from the old production dir into staging before the atomic swap.
var preserveDirs = []string{"repo-rpm"}

// StagePublish atomically swaps staging to production with backup.
func StagePublish(ctx context.Context, repoRoot, productName, stagingDir string, rollbackKeep int, log *LogWriter) error {
	productionDir := filepath.Join(repoRoot, productName)

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

		// Move existing .rollback dir to staging so it survives the swap
		oldRollback := filepath.Join(productionDir, ".rollback")
		stagingRollback := filepath.Join(stagingDir, ".rollback")
		if info, statErr := os.Stat(oldRollback); statErr == nil && info.IsDir() {
			if err := os.Rename(oldRollback, stagingRollback); err != nil {
				log.WriteLog("Warning: failed to preserve rollback history: %v", err)
			}
		}
		os.MkdirAll(stagingRollback, 0755)

		// Create backup snapshot: copy production (minus .rollback) into staging/.rollback/<timestamp>
		backupDir := filepath.Join(stagingRollback, timestamp)
		log.WriteLog("Backing up %s to .rollback/%s", productionDir, timestamp)
		if err := copyDirExcluding(productionDir, backupDir, ".rollback"); err != nil {
			log.WriteLog("Warning: backup failed: %v", err)
		}

		// Remove old production dir to make way for atomic swap
		if err := os.RemoveAll(productionDir); err != nil {
			return fmt.Errorf("remove old production dir: %w", err)
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

	// Clean old backups (keep latest N)
	if rollbackKeep <= 0 {
		rollbackKeep = 3
	}
	cleanOldBackups(filepath.Join(productionDir, ".rollback"), rollbackKeep, log)

	// Clean old repo-rpm files (keep only the latest per name prefix)
	cleanOldRepoRPMs(filepath.Join(productionDir, "repo-rpm"), log)

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

// cleanOldRepoRPMs removes old repo RPM files, keeping only the latest 3.
func cleanOldRepoRPMs(dir string, log *LogWriter) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	type rpmFile struct {
		name    string
		modTime int64
	}

	var rpms []rpmFile
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".rpm" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		rpms = append(rpms, rpmFile{name: e.Name(), modTime: info.ModTime().UnixNano()})
	}

	if len(rpms) <= 3 {
		return
	}

	// Sort by mod time ascending (oldest first)
	sort.Slice(rpms, func(i, j int) bool {
		return rpms[i].modTime < rpms[j].modTime
	})

	// Remove all but the latest 3
	for _, f := range rpms[:len(rpms)-3] {
		path := filepath.Join(dir, f.name)
		log.WriteLog("Removing old repo RPM: %s", f.name)
		os.Remove(path)
	}
}

// copyDirExcluding recursively copies a directory tree, excluding a named subdirectory.
func copyDirExcluding(src, dst, exclude string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(src, path)
		if relPath == exclude || strings.HasPrefix(relPath, exclude+string(filepath.Separator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
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
