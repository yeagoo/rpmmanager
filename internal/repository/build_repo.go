package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ivmm/rpmmanager/internal/models"
)

type BuildRepo struct {
	db *sql.DB
}

func NewBuildRepo(db *sql.DB) *BuildRepo {
	return &BuildRepo{db: db}
}

func (r *BuildRepo) Create(b *models.Build) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO builds (product_id, version, status, trigger_type, target_distros, architectures, log_file)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.ProductID, b.Version, b.Status, b.TriggerType,
		b.TargetDistrosJSON(), b.ArchitecturesJSON(), b.LogFile,
	)
	if err != nil {
		return 0, fmt.Errorf("insert build: %w", err)
	}
	return result.LastInsertId()
}

func (r *BuildRepo) GetByID(id int64) (*models.Build, error) {
	b := &models.Build{}
	var targetDistros, architectures string
	var startedAt, finishedAt sql.NullTime
	err := r.db.QueryRow(`
		SELECT b.id, b.product_id, b.version, b.status, b.current_stage, b.trigger_type,
			b.target_distros, b.architectures, b.rpm_count, b.symlink_count,
			b.error_message, b.log_file, b.started_at, b.finished_at, b.duration_seconds, b.created_at,
			p.name, p.display_name
		FROM builds b
		JOIN products p ON p.id = b.product_id
		WHERE b.id = ?`, id).Scan(
		&b.ID, &b.ProductID, &b.Version, &b.Status, &b.CurrentStage, &b.TriggerType,
		&targetDistros, &architectures, &b.RPMCount, &b.SymlinkCount,
		&b.ErrorMessage, &b.LogFile, &startedAt, &finishedAt, &b.DurationSeconds, &b.CreatedAt,
		&b.ProductName, &b.ProductDisplayName,
	)
	if err != nil {
		return nil, fmt.Errorf("get build %d: %w", id, err)
	}
	b.TargetDistros = models.ParseJSONStringArray(targetDistros)
	b.Architectures = models.ParseJSONStringArray(architectures)
	if startedAt.Valid {
		b.StartedAt = &startedAt.Time
	}
	if finishedAt.Valid {
		b.FinishedAt = &finishedAt.Time
	}
	return b, nil
}

func (r *BuildRepo) List(productID int64, limit int) ([]models.Build, error) {
	query := `
		SELECT b.id, b.product_id, b.version, b.status, b.current_stage, b.trigger_type,
			b.target_distros, b.architectures, b.rpm_count, b.symlink_count,
			b.error_message, b.log_file, b.started_at, b.finished_at, b.duration_seconds, b.created_at,
			p.name, p.display_name
		FROM builds b
		JOIN products p ON p.id = b.product_id`

	var args []interface{}
	if productID > 0 {
		query += " WHERE b.product_id = ?"
		args = append(args, productID)
	}
	query += " ORDER BY b.created_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list builds: %w", err)
	}
	defer rows.Close()

	var builds []models.Build
	for rows.Next() {
		var b models.Build
		var targetDistros, architectures string
		var startedAt, finishedAt sql.NullTime
		err := rows.Scan(
			&b.ID, &b.ProductID, &b.Version, &b.Status, &b.CurrentStage, &b.TriggerType,
			&targetDistros, &architectures, &b.RPMCount, &b.SymlinkCount,
			&b.ErrorMessage, &b.LogFile, &startedAt, &finishedAt, &b.DurationSeconds, &b.CreatedAt,
			&b.ProductName, &b.ProductDisplayName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan build: %w", err)
		}
		b.TargetDistros = models.ParseJSONStringArray(targetDistros)
		b.Architectures = models.ParseJSONStringArray(architectures)
		if startedAt.Valid {
			b.StartedAt = &startedAt.Time
		}
		if finishedAt.Valid {
			b.FinishedAt = &finishedAt.Time
		}
		builds = append(builds, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate builds: %w", err)
	}
	return builds, nil
}

func (r *BuildRepo) UpdateStatus(id int64, status models.BuildStatus, stage string) error {
	_, err := r.db.Exec(`UPDATE builds SET status = ?, current_stage = ? WHERE id = ?`, status, stage, id)
	return err
}

func (r *BuildRepo) UpdateStarted(id int64) error {
	now := time.Now()
	_, err := r.db.Exec(`UPDATE builds SET started_at = ? WHERE id = ?`, now, id)
	return err
}

func (r *BuildRepo) UpdateFinished(id int64, status models.BuildStatus, errMsg string, rpmCount, symlinkCount int) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE builds SET status = ?, error_message = ?, finished_at = ?, rpm_count = ?, symlink_count = ?,
			duration_seconds = CAST((julianday(?) - julianday(started_at)) * 86400 AS INTEGER)
		WHERE id = ?`, status, errMsg, now, rpmCount, symlinkCount, now, id)
	return err
}

func (r *BuildRepo) HasRunningBuild(productID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM builds
		WHERE product_id = ? AND status IN ('pending', 'building', 'signing', 'publishing', 'verifying')`,
		productID).Scan(&count)
	return count > 0, err
}

