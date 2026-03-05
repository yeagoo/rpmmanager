package repository

import (
	"database/sql"
	"fmt"

	"github.com/ivmm/rpmmanager/internal/models"
)

type MonitorRepo struct {
	db *sql.DB
}

func NewMonitorRepo(db *sql.DB) *MonitorRepo {
	return &MonitorRepo{db: db}
}

func (r *MonitorRepo) GetByProductID(productID int64) (*models.Monitor, error) {
	m := &models.Monitor{}
	var lastChecked sql.NullTime
	err := r.db.QueryRow(`
		SELECT m.id, m.product_id, m.enabled, m.check_interval, m.auto_build,
			m.last_checked_at, m.last_known_version, m.last_error, m.created_at, m.updated_at,
			p.name, p.display_name, p.source_type, p.source_github_owner, p.source_github_repo
		FROM monitors m
		JOIN products p ON p.id = m.product_id
		WHERE m.product_id = ?`, productID).Scan(
		&m.ID, &m.ProductID, &m.Enabled, &m.CheckInterval, &m.AutoBuild,
		&lastChecked, &m.LastKnownVersion, &m.LastError, &m.CreatedAt, &m.UpdatedAt,
		&m.ProductName, &m.ProductDisplayName, &m.SourceType, &m.SourceGithubOwner, &m.SourceGithubRepo,
	)
	if err != nil {
		return nil, err
	}
	if lastChecked.Valid {
		m.LastCheckedAt = &lastChecked.Time
	}
	return m, nil
}

func (r *MonitorRepo) List() ([]models.Monitor, error) {
	rows, err := r.db.Query(`
		SELECT m.id, m.product_id, m.enabled, m.check_interval, m.auto_build,
			m.last_checked_at, m.last_known_version, m.last_error, m.created_at, m.updated_at,
			p.name, p.display_name, p.source_type, p.source_github_owner, p.source_github_repo
		FROM monitors m
		JOIN products p ON p.id = m.product_id
		ORDER BY p.display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []models.Monitor
	for rows.Next() {
		var m models.Monitor
		var lastChecked sql.NullTime
		err := rows.Scan(
			&m.ID, &m.ProductID, &m.Enabled, &m.CheckInterval, &m.AutoBuild,
			&lastChecked, &m.LastKnownVersion, &m.LastError, &m.CreatedAt, &m.UpdatedAt,
			&m.ProductName, &m.ProductDisplayName, &m.SourceType, &m.SourceGithubOwner, &m.SourceGithubRepo,
		)
		if err != nil {
			return nil, fmt.Errorf("scan monitor: %w", err)
		}
		if lastChecked.Valid {
			m.LastCheckedAt = &lastChecked.Time
		}
		monitors = append(monitors, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitors: %w", err)
	}
	return monitors, nil
}

func (r *MonitorRepo) Upsert(productID int64) error {
	_, err := r.db.Exec(`
		INSERT INTO monitors (product_id) VALUES (?)
		ON CONFLICT(product_id) DO NOTHING`, productID)
	return err
}

func (r *MonitorRepo) Update(productID int64, enabled *bool, checkInterval string, autoBuild *bool) error {
	if enabled != nil {
		if _, err := r.db.Exec("UPDATE monitors SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE product_id = ?", *enabled, productID); err != nil {
			return fmt.Errorf("update monitor enabled: %w", err)
		}
	}
	if checkInterval != "" {
		if _, err := r.db.Exec("UPDATE monitors SET check_interval = ?, updated_at = CURRENT_TIMESTAMP WHERE product_id = ?", checkInterval, productID); err != nil {
			return fmt.Errorf("update monitor interval: %w", err)
		}
	}
	if autoBuild != nil {
		if _, err := r.db.Exec("UPDATE monitors SET auto_build = ?, updated_at = CURRENT_TIMESTAMP WHERE product_id = ?", *autoBuild, productID); err != nil {
			return fmt.Errorf("update monitor auto_build: %w", err)
		}
	}
	return nil
}

func (r *MonitorRepo) UpdateCheckResult(productID int64, version, errMsg string) error {
	_, err := r.db.Exec(`
		UPDATE monitors SET last_checked_at = CURRENT_TIMESTAMP, last_known_version = ?, last_error = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE product_id = ?`, version, errMsg, productID)
	return err
}

func (r *MonitorRepo) GetEnabled() ([]models.Monitor, error) {
	rows, err := r.db.Query(`
		SELECT m.id, m.product_id, m.enabled, m.check_interval, m.auto_build,
			m.last_checked_at, m.last_known_version, m.last_error, m.created_at, m.updated_at,
			p.name, p.display_name, p.source_type, p.source_github_owner, p.source_github_repo
		FROM monitors m
		JOIN products p ON p.id = m.product_id
		WHERE m.enabled = TRUE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []models.Monitor
	for rows.Next() {
		var m models.Monitor
		var lastChecked sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.ProductID, &m.Enabled, &m.CheckInterval, &m.AutoBuild,
			&lastChecked, &m.LastKnownVersion, &m.LastError, &m.CreatedAt, &m.UpdatedAt,
			&m.ProductName, &m.ProductDisplayName, &m.SourceType, &m.SourceGithubOwner, &m.SourceGithubRepo,
		); err != nil {
			return nil, fmt.Errorf("scan enabled monitor: %w", err)
		}
		if lastChecked.Valid {
			m.LastCheckedAt = &lastChecked.Time
		}
		monitors = append(monitors, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled monitors: %w", err)
	}
	return monitors, nil
}
