package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ivmm/rpmmanager/internal/distromap"
)

// GenerateRepoTemplates creates .repo files for each distro:version in the templates directory.
func GenerateRepoTemplates(baseDir, productName, baseURL string, distroVersions []string, customLines []distromap.ProductLine, logWriter *LogWriter) error {
	templatesDir := filepath.Join(baseDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("create templates dir: %w", err)
	}

	for _, dv := range distroVersions {
		parsed := distromap.ParseDistroVersion(dv)
		pl, err := distromap.Resolve(dv, customLines)
		if err != nil {
			logWriter.WriteLog("Warning: skipping .repo for %s: %s", dv, err)
			continue
		}

		repoID := fmt.Sprintf("%s-%s-%s", productName, parsed.Distro, parsed.Version)
		repoName := fmt.Sprintf("%s for %s %s", productName, parsed.Distro, parsed.Version)

		// Build baseurl path
		baseurlPath := fmt.Sprintf("%s/%s/%s/$basearch/", baseURL, productName, pl.Path)

		content := fmt.Sprintf(`[%s]
name=%s
baseurl=%s
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=%s/%s/gpg.key
`, repoID, repoName, baseurlPath, baseURL, productName)

		fileName := fmt.Sprintf("%s.repo", repoID)
		filePath := filepath.Join(templatesDir, fileName)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write repo template %s: %w", fileName, err)
		}
		logWriter.WriteLog("Generated: templates/%s", fileName)
	}

	return nil
}
