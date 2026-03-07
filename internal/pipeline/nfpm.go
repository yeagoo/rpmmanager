package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ivmm/rpmmanager/internal/distromap"
	"github.com/ivmm/rpmmanager/internal/models"
	"gopkg.in/yaml.v3"
)

// NfpmConfigData represents the user-defined nfpm packaging fields stored in product.nfpm_config.
type NfpmConfigData struct {
	Description string            `json:"description"`
	Contents    []NfpmContentItem `json:"contents"`
	Depends     []string          `json:"depends"`
	RPMGroup    string            `json:"rpm_group"`
}

type NfpmContentItem struct {
	Src  string `json:"src" yaml:"src,omitempty"`
	Dst  string `json:"dst" yaml:"dst"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	Mode string `json:"mode,omitempty" yaml:"file_info,omitempty"`
}

// GenerateNfpmYAML creates a temporary nfpm YAML config file for the given product, version, arch, and product line.
func GenerateNfpmYAML(
	product *models.Product,
	version string,
	arch string,
	pl *distromap.ProductLine,
	binaryPath string,
	tempDir string,
) (string, error) {
	// Parse user-defined nfpm config
	var nfpmData NfpmConfigData
	if err := json.Unmarshal([]byte(product.NfpmConfig), &nfpmData); err != nil {
		return "", fmt.Errorf("parse nfpm_config: %w", err)
	}

	// Write systemd service file to temp
	var systemdPath string
	if product.SystemdService != "" {
		systemdPath = filepath.Join(tempDir, product.Name+".service")
		if err := os.WriteFile(systemdPath, []byte(product.SystemdService), 0644); err != nil {
			return "", fmt.Errorf("write systemd service: %w", err)
		}
	}

	// Write default config to temp
	var configPath string
	if product.DefaultConfig != "" && product.DefaultConfigPath != "" {
		configPath = filepath.Join(tempDir, filepath.Base(product.DefaultConfigPath))
		if err := os.WriteFile(configPath, []byte(product.DefaultConfig), 0644); err != nil {
			return "", fmt.Errorf("write default config: %w", err)
		}
	}

	// Write scripts to temp
	var postinstallPath, preremovePath string
	if product.ScriptPostinstall != "" {
		postinstallPath = filepath.Join(tempDir, "postinstall.sh")
		os.WriteFile(postinstallPath, []byte(product.ScriptPostinstall), 0755)
	}
	if product.ScriptPreremove != "" {
		preremovePath = filepath.Join(tempDir, "preremove.sh")
		os.WriteFile(preremovePath, []byte(product.ScriptPreremove), 0755)
	}

	// Build contents list by resolving template variables
	var contents []map[string]interface{}
	for _, c := range nfpmData.Contents {
		src := c.Src
		src = strings.ReplaceAll(src, "{{binary}}", binaryPath)
		if systemdPath != "" {
			src = strings.ReplaceAll(src, "{{systemd_service}}", systemdPath)
		}
		if configPath != "" {
			src = strings.ReplaceAll(src, "{{default_config}}", configPath)
		}

		item := map[string]interface{}{
			"dst": c.Dst,
		}
		if src != "" {
			item["src"] = src
		}
		if c.Type != "" {
			item["type"] = c.Type
		}
		if c.Mode != "" {
			if mode, err := strconv.ParseUint(c.Mode, 8, 32); err == nil {
				item["file_info"] = map[string]interface{}{"mode": mode}
			}
		}
		contents = append(contents, item)
	}

	// Map architecture names
	nfpmArch := arch
	if arch == "x86_64" {
		nfpmArch = "amd64"
	} else if arch == "aarch64" {
		nfpmArch = "arm64"
	}

	// Build nfpm config
	release := fmt.Sprintf("1.%s", pl.Tag)
	config := map[string]interface{}{
		"name":        product.Name,
		"arch":        nfpmArch,
		"version":     version,
		"release":     release,
		"maintainer":  product.Maintainer,
		"description": nfpmData.Description,
		"vendor":      product.Vendor,
		"homepage":    product.Homepage,
		"license":     product.License,
		"contents":    contents,
		"rpm": map[string]interface{}{
			"compression": pl.Compression,
			"group":       nfpmData.RPMGroup,
		},
	}

	if len(nfpmData.Depends) > 0 {
		config["depends"] = nfpmData.Depends
	}

	// Scripts
	scripts := make(map[string]interface{})
	if postinstallPath != "" {
		scripts["postinstall"] = postinstallPath
	}
	if preremovePath != "" {
		scripts["preremove"] = preremovePath
	}
	if len(scripts) > 0 {
		config["scripts"] = scripts
	}

	// Write YAML
	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal nfpm yaml: %w", err)
	}

	configFile := filepath.Join(tempDir, fmt.Sprintf("nfpm-%s-%s-%s.yaml", product.Name, pl.ID, arch))
	if err := os.WriteFile(configFile, yamlBytes, 0644); err != nil {
		return "", fmt.Errorf("write nfpm config: %w", err)
	}

	return configFile, nil
}
