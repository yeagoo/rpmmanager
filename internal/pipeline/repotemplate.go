package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ivmm/rpmmanager/internal/distromap"
)

// GenerateRepoTemplates creates .repo files for each distro:version in the templates directory.
func GenerateRepoTemplates(baseDir, productName, baseURL string, distroVersions []string, customLines []distromap.ProductLine, logWriter *LogWriter) error {
	templatesDir := filepath.Join(baseDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("create templates dir: %w", err)
	}

	var repoFiles []string

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
		repoFiles = append(repoFiles, fileName)
		logWriter.WriteLog("Generated: templates/%s", fileName)
	}

	// Generate index.html for directory browsing
	if err := generateTemplatesIndex(templatesDir, productName, baseURL, repoFiles); err != nil {
		logWriter.WriteLog("Warning: failed to generate templates/index.html: %s", err)
	}

	return nil
}

func generateTemplatesIndex(templatesDir, productName, baseURL string, repoFiles []string) error {
	var rows strings.Builder
	for _, f := range repoFiles {
		rows.WriteString(fmt.Sprintf("      <tr><td><a href=\"%s\">%s</a></td><td><code>curl -o /etc/yum.repos.d/%s %s/%s/templates/%s</code></td></tr>\n",
			f, f, f, baseURL, productName, f))
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>%s - Repository Templates</title>
  <style>
    body { font-family: -apple-system, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; color: #333; }
    h1 { border-bottom: 1px solid #eee; padding-bottom: 10px; }
    table { border-collapse: collapse; width: 100%%; }
    th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #eee; }
    th { background: #f8f8f8; }
    a { color: #0366d6; text-decoration: none; }
    a:hover { text-decoration: underline; }
    code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; font-size: 13px; }
  </style>
</head>
<body>
  <h1>%s - Repository Templates</h1>
  <p>Download a <code>.repo</code> file and place it in <code>/etc/yum.repos.d/</code> to enable this repository.</p>
  <table>
    <thead><tr><th>File</th><th>Quick Install</th></tr></thead>
    <tbody>
%s    </tbody>
  </table>
</body>
</html>
`, productName, productName, rows.String())

	return os.WriteFile(filepath.Join(templatesDir, "index.html"), []byte(html), 0644)
}
