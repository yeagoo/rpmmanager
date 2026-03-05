package models

import "time"

type BuildStatus string

const (
	BuildStatusPending    BuildStatus = "pending"
	BuildStatusBuilding   BuildStatus = "building"
	BuildStatusSigning    BuildStatus = "signing"
	BuildStatusPublishing BuildStatus = "publishing"
	BuildStatusVerifying  BuildStatus = "verifying"
	BuildStatusSuccess    BuildStatus = "success"
	BuildStatusFailed     BuildStatus = "failed"
	BuildStatusCancelled  BuildStatus = "cancelled"
)

type Build struct {
	ID              int64       `json:"id"`
	ProductID       int64       `json:"product_id"`
	Version         string      `json:"version"`
	Status          BuildStatus `json:"status"`
	CurrentStage    string      `json:"current_stage"`
	TriggerType     string      `json:"trigger_type"`
	TargetDistros   []string    `json:"target_distros"`
	Architectures   []string    `json:"architectures"`
	RPMCount        int         `json:"rpm_count"`
	SymlinkCount    int         `json:"symlink_count"`
	ErrorMessage    string      `json:"error_message"`
	LogFile         string      `json:"log_file"`
	StartedAt       *time.Time  `json:"started_at"`
	FinishedAt      *time.Time  `json:"finished_at"`
	DurationSeconds int         `json:"duration_seconds"`
	CreatedAt       time.Time   `json:"created_at"`

	// Joined fields
	ProductName        string `json:"product_name,omitempty"`
	ProductDisplayName string `json:"product_display_name,omitempty"`
}

func (b *Build) TargetDistrosJSON() string {
	return StringSliceToJSON(b.TargetDistros)
}

func (b *Build) ArchitecturesJSON() string {
	return StringSliceToJSON(b.Architectures)
}

type TriggerBuildRequest struct {
	ProductID int64  `json:"product_id"`
	Version   string `json:"version"`
}
