package models

import "time"

type Monitor struct {
	ID               int64      `json:"id"`
	ProductID        int64      `json:"product_id"`
	Enabled          bool       `json:"enabled"`
	CheckInterval    string     `json:"check_interval"`
	AutoBuild        bool       `json:"auto_build"`
	LastCheckedAt    *time.Time `json:"last_checked_at"`
	LastKnownVersion string     `json:"last_known_version"`
	LastError        string     `json:"last_error"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`

	// Joined fields
	ProductName        string `json:"product_name,omitempty"`
	ProductDisplayName string `json:"product_display_name,omitempty"`
	SourceType         string `json:"source_type,omitempty"`
	SourceGithubOwner  string `json:"source_github_owner,omitempty"`
	SourceGithubRepo   string `json:"source_github_repo,omitempty"`
}

type UpdateMonitorRequest struct {
	Enabled       *bool  `json:"enabled,omitempty"`
	CheckInterval string `json:"check_interval,omitempty"`
	AutoBuild     *bool  `json:"auto_build,omitempty"`
}
