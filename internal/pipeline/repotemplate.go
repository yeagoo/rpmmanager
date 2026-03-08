package pipeline

import (
	"bytes"
	"fmt"
	"html/template"
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

type repoFileEntry struct {
	FileName string
	CurlCmd  string
}

type templatesIndexData struct {
	ProductName string
	Files       []repoFileEntry
}

var templatesIndexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>{{.ProductName}} - Repository Templates</title>
  <style>
    body { font-family: -apple-system, sans-serif; max-width: 900px; margin: 40px auto; padding: 0 20px; color: #333; }
    h1 { border-bottom: 1px solid #eee; padding-bottom: 10px; }
    table { border-collapse: collapse; width: 100%; }
    th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #eee; }
    th { background: #f8f8f8; }
    a { color: #0366d6; text-decoration: none; }
    a:hover { text-decoration: underline; }
    code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; font-size: 13px; }
  </style>
</head>
<body>
  <h1>{{.ProductName}} - Repository Templates</h1>
  <p>Download a <code>.repo</code> file and place it in <code>/etc/yum.repos.d/</code> to enable this repository.</p>
  <table>
    <thead><tr><th>File</th><th>Quick Install</th></tr></thead>
    <tbody>
{{- range .Files}}
      <tr><td><a href="{{.FileName}}">{{.FileName}}</a></td><td><code>{{.CurlCmd}}</code></td></tr>
{{- end}}
    </tbody>
  </table>
</body>
</html>
`))

func generateTemplatesIndex(templatesDir, productName, baseURL string, repoFiles []string) error {
	var files []repoFileEntry
	for _, f := range repoFiles {
		files = append(files, repoFileEntry{
			FileName: f,
			CurlCmd:  fmt.Sprintf("curl -o /etc/yum.repos.d/%s %s/%s/templates/%s", f, baseURL, productName, f),
		})
	}

	var buf bytes.Buffer
	if err := templatesIndexTmpl.Execute(&buf, templatesIndexData{
		ProductName: productName,
		Files:       files,
	}); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return os.WriteFile(filepath.Join(templatesDir, "index.html"), buf.Bytes(), 0644)
}
