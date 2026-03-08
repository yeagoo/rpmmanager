package models

import (
	"encoding/json"
	"time"
)

type Product struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	DisplayName       string    `json:"display_name"`
	Description       string    `json:"description"`
	SourceType        string    `json:"source_type"`
	SourceGithubOwner string    `json:"source_github_owner"`
	SourceGithubRepo         string    `json:"source_github_repo"`
	SourceGithubAssetPattern string    `json:"source_github_asset_pattern"`
	SourceURLTemplate        string    `json:"source_url_template"`
	NfpmConfig        string    `json:"nfpm_config"`
	TargetDistros     []string  `json:"target_distros"`
	Architectures     []string  `json:"architectures"`
	ProductLines      string    `json:"product_lines,omitempty"`
	Maintainer        string    `json:"maintainer"`
	Vendor            string    `json:"vendor"`
	Homepage          string    `json:"homepage"`
	License           string    `json:"license"`
	ScriptPostinstall string    `json:"script_postinstall"`
	ScriptPreremove   string    `json:"script_preremove"`
	SystemdService    string    `json:"systemd_service"`
	DefaultConfig     string    `json:"default_config"`
	DefaultConfigPath string    `json:"default_config_path"`
	ExtraFiles        string    `json:"extra_files"`
	GPGKeyID          *int64    `json:"gpg_key_id"`
	BaseURL           string    `json:"base_url"`
	SM2Enabled        bool      `json:"sm2_enabled"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TargetDistrosJSON returns the target_distros as a JSON string for DB storage.
func (p *Product) TargetDistrosJSON() string {
	b, _ := json.Marshal(p.TargetDistros)
	return string(b)
}

// ArchitecturesJSON returns the architectures as a JSON string for DB storage.
func (p *Product) ArchitecturesJSON() string {
	b, _ := json.Marshal(p.Architectures)
	return string(b)
}

// ParseJSONStringArray parses a JSON array string into []string.
func ParseJSONStringArray(s string) []string {
	var result []string
	json.Unmarshal([]byte(s), &result)
	return result
}

// StringSliceToJSON converts a string slice to a JSON string.
func StringSliceToJSON(s []string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

type ProductListItem struct {
	Product
	LatestVersion string `json:"latest_version,omitempty"`
	LastBuildAt   string `json:"last_build_at,omitempty"`
}
