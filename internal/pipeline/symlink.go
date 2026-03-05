package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ivmm/rpmmanager/internal/distromap"
)

// CreateSymlinks creates distro alias symlinks in the staging directory.
// Returns the number of symlinks created.
func CreateSymlinks(baseDir string, distroVersions []string, customLines []distromap.ProductLine, logWriter *LogWriter) (int, error) {
	links, err := distromap.SymlinksForDistros(distroVersions, customLines)
	if err != nil {
		return 0, err
	}

	count := 0
	for linkPath, target := range links {
		fullPath := filepath.Join(baseDir, linkPath)

		// Remove existing symlink
		os.Remove(fullPath)

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return count, fmt.Errorf("create symlink parent %s: %w", filepath.Dir(fullPath), err)
		}

		if err := os.Symlink(target, fullPath); err != nil {
			return count, fmt.Errorf("create symlink %s -> %s: %w", linkPath, target, err)
		}
		logWriter.WriteLog("Symlink: %s -> %s", linkPath, target)
		count++
	}

	return count, nil
}
