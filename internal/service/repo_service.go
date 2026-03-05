package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ivmm/rpmmanager/internal/config"
)

// productNameRegex validates product names to prevent path traversal.
var productNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*[a-z0-9]$`)

type RepoService struct {
	cfg     *config.Config
	absRoot string // absolute, cleaned path to repo root
}

func NewRepoService(cfg *config.Config) *RepoService {
	absRoot, _ := filepath.Abs(cfg.Storage.RepoRoot)
	return &RepoService{cfg: cfg, absRoot: absRoot}
}

type RepoEntry struct {
	Name    string      `json:"name"`
	Path    string      `json:"path"`
	IsDir   bool        `json:"is_dir"`
	Size    int64       `json:"size"`
	ModTime time.Time   `json:"mod_time"`
	Items   []RepoEntry `json:"items,omitempty"`
}

type RepoInfo struct {
	Product   string `json:"product"`
	Path      string `json:"path"`
	TotalSize int64  `json:"total_size"`
	FileCount int    `json:"file_count"`
	DirCount  int    `json:"dir_count"`
	RPMCount  int    `json:"rpm_count"`
	HasRepoMD bool   `json:"has_repomd"`
}

// validateProduct ensures the product name is safe for filesystem operations.
func validateProduct(product string) error {
	if product == "" || !productNameRegex.MatchString(product) {
		return fmt.Errorf("invalid product name")
	}
	if strings.Contains(product, "..") {
		return fmt.Errorf("invalid product name")
	}
	return nil
}

// safePath joins a relative path to the repo root and validates it doesn't escape.
func (s *RepoService) safePath(relPath string) (string, error) {
	absPath := filepath.Join(s.absRoot, filepath.Clean(relPath))
	// Ensure path is within repo root (use trailing separator to prevent prefix attack)
	if !strings.HasPrefix(absPath, s.absRoot+string(filepath.Separator)) && absPath != s.absRoot {
		return "", fmt.Errorf("invalid path")
	}
	return absPath, nil
}

// ListProducts returns top-level product directories in the repo root.
func (s *RepoService) ListProducts() ([]RepoInfo, error) {
	entries, err := os.ReadDir(s.absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []RepoInfo{}, nil
		}
		return nil, err
	}

	var repos []RepoInfo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info := RepoInfo{
			Product: e.Name(),
			Path:    e.Name(),
		}
		productDir := filepath.Join(s.absRoot, e.Name())
		info.TotalSize, info.FileCount, info.DirCount, info.RPMCount = s.countDir(productDir)
		info.HasRepoMD = s.hasRepoMD(productDir)
		repos = append(repos, info)
	}
	return repos, nil
}

// GetTree returns a directory tree for a given path relative to repo root.
func (s *RepoService) GetTree(relPath string, depth int) ([]RepoEntry, error) {
	absPath, err := s.safePath(relPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var result []RepoEntry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		entry := RepoEntry{
			Name:    e.Name(),
			Path:    filepath.Join(relPath, e.Name()),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		if e.IsDir() && depth > 1 {
			entry.Items, _ = s.GetTree(entry.Path, depth-1)
		}
		result = append(result, entry)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

// GetFileContent returns the content of a file (for .repo files, repomd.xml, etc).
func (s *RepoService) GetFileContent(relPath string) ([]byte, error) {
	absPath, err := s.safePath(relPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if info.Size() > 1<<20 {
		return nil, fmt.Errorf("file too large")
	}
	return os.ReadFile(absPath)
}

// ListRollbacks returns available rollback snapshots for a product.
func (s *RepoService) ListRollbacks(product string) ([]string, error) {
	if err := validateProduct(product); err != nil {
		return nil, err
	}
	rollbackDir := filepath.Join(s.absRoot, product, ".rollback")
	entries, err := os.ReadDir(rollbackDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var snapshots []string
	for _, e := range entries {
		if e.IsDir() {
			snapshots = append(snapshots, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(snapshots)))
	return snapshots, nil
}

// Rollback restores a product repo from a rollback snapshot.
func (s *RepoService) Rollback(product, snapshot string) error {
	if err := validateProduct(product); err != nil {
		return err
	}
	// Validate snapshot name (should be a timestamp-like string, no path separators)
	if strings.ContainsAny(snapshot, "/\\..") {
		return fmt.Errorf("invalid snapshot name")
	}

	productDir := filepath.Join(s.absRoot, product)
	rollbackSrc := filepath.Join(productDir, ".rollback", snapshot)

	if _, err := os.Stat(rollbackSrc); err != nil {
		return fmt.Errorf("rollback snapshot not found: %s", snapshot)
	}

	// Move current to temp backup
	tempBackup := productDir + ".rollback-tmp"
	if err := os.Rename(productDir, tempBackup); err != nil {
		return fmt.Errorf("backup current: %w", err)
	}

	// Move rollback snapshot to current
	if err := os.Rename(rollbackSrc, productDir); err != nil {
		// Try to restore
		os.Rename(tempBackup, productDir)
		return fmt.Errorf("restore rollback: %w", err)
	}

	// Restore .rollback dir from temp backup
	oldRollback := filepath.Join(tempBackup, ".rollback")
	newRollback := filepath.Join(productDir, ".rollback")
	if err := os.Rename(oldRollback, newRollback); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to restore rollback directory: %v\n", err)
	}
	if err := os.RemoveAll(tempBackup); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to clean up temp backup: %v\n", err)
	}

	return nil
}

func (s *RepoService) countDir(dir string) (totalSize int64, fileCount, dirCount, rpmCount int) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			dirCount++
		} else {
			fileCount++
			totalSize += info.Size()
			if strings.HasSuffix(info.Name(), ".rpm") {
				rpmCount++
			}
		}
		return nil
	})
	return
}

func (s *RepoService) hasRepoMD(dir string) bool {
	found := false
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == "repomd.xml" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
