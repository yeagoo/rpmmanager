package repository

import (
	"database/sql"
	"fmt"

	"github.com/ivmm/rpmmanager/internal/models"
)

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {
	return &ProductRepo{db: db}
}

func (r *ProductRepo) Create(p *models.Product) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO products (
			name, display_name, description,
			source_type, source_github_owner, source_github_repo, source_url_template,
			nfpm_config, target_distros, architectures, product_lines,
			maintainer, vendor, homepage, license,
			script_postinstall, script_preremove,
			systemd_service, default_config, default_config_path,
			extra_files, gpg_key_id, base_url, sm2_enabled, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.DisplayName, p.Description,
		p.SourceType, p.SourceGithubOwner, p.SourceGithubRepo, p.SourceURLTemplate,
		p.NfpmConfig, p.TargetDistrosJSON(), p.ArchitecturesJSON(), nilIfEmpty(p.ProductLines),
		p.Maintainer, p.Vendor, p.Homepage, p.License,
		p.ScriptPostinstall, p.ScriptPreremove,
		p.SystemdService, p.DefaultConfig, p.DefaultConfigPath,
		p.ExtraFiles, p.GPGKeyID, p.BaseURL, p.SM2Enabled, p.Enabled,
	)
	if err != nil {
		return 0, fmt.Errorf("insert product: %w", err)
	}
	return result.LastInsertId()
}

func (r *ProductRepo) GetByName(name string) (*models.Product, error) {
	p := &models.Product{}
	var targetDistros, architectures string
	var productLines sql.NullString
	err := r.db.QueryRow(`SELECT
		id, name, display_name, description,
		source_type, source_github_owner, source_github_repo, source_url_template,
		nfpm_config, target_distros, architectures, product_lines,
		maintainer, vendor, homepage, license,
		script_postinstall, script_preremove,
		systemd_service, default_config, default_config_path,
		extra_files, gpg_key_id, base_url, sm2_enabled, enabled,
		created_at, updated_at
		FROM products WHERE name = ?`, name).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description,
		&p.SourceType, &p.SourceGithubOwner, &p.SourceGithubRepo, &p.SourceURLTemplate,
		&p.NfpmConfig, &targetDistros, &architectures, &productLines,
		&p.Maintainer, &p.Vendor, &p.Homepage, &p.License,
		&p.ScriptPostinstall, &p.ScriptPreremove,
		&p.SystemdService, &p.DefaultConfig, &p.DefaultConfigPath,
		&p.ExtraFiles, &p.GPGKeyID, &p.BaseURL, &p.SM2Enabled, &p.Enabled,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get product by name %s: %w", name, err)
	}
	p.TargetDistros = models.ParseJSONStringArray(targetDistros)
	p.Architectures = models.ParseJSONStringArray(architectures)
	if productLines.Valid {
		p.ProductLines = productLines.String
	}
	return p, nil
}

func (r *ProductRepo) GetByID(id int64) (*models.Product, error) {
	p := &models.Product{}
	var targetDistros, architectures string
	var productLines sql.NullString
	err := r.db.QueryRow(`SELECT
		id, name, display_name, description,
		source_type, source_github_owner, source_github_repo, source_url_template,
		nfpm_config, target_distros, architectures, product_lines,
		maintainer, vendor, homepage, license,
		script_postinstall, script_preremove,
		systemd_service, default_config, default_config_path,
		extra_files, gpg_key_id, base_url, sm2_enabled, enabled,
		created_at, updated_at
		FROM products WHERE id = ?`, id).Scan(
		&p.ID, &p.Name, &p.DisplayName, &p.Description,
		&p.SourceType, &p.SourceGithubOwner, &p.SourceGithubRepo, &p.SourceURLTemplate,
		&p.NfpmConfig, &targetDistros, &architectures, &productLines,
		&p.Maintainer, &p.Vendor, &p.Homepage, &p.License,
		&p.ScriptPostinstall, &p.ScriptPreremove,
		&p.SystemdService, &p.DefaultConfig, &p.DefaultConfigPath,
		&p.ExtraFiles, &p.GPGKeyID, &p.BaseURL, &p.SM2Enabled, &p.Enabled,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get product %d: %w", id, err)
	}
	p.TargetDistros = models.ParseJSONStringArray(targetDistros)
	p.Architectures = models.ParseJSONStringArray(architectures)
	if productLines.Valid {
		p.ProductLines = productLines.String
	}
	return p, nil
}

func (r *ProductRepo) List() ([]models.ProductListItem, error) {
	rows, err := r.db.Query(`
		SELECT
			p.id, p.name, p.display_name, p.description,
			p.source_type, p.source_github_owner, p.source_github_repo, p.source_url_template,
			p.nfpm_config, p.target_distros, p.architectures, p.product_lines,
			p.maintainer, p.vendor, p.homepage, p.license,
			p.script_postinstall, p.script_preremove,
			p.systemd_service, p.default_config, p.default_config_path,
			p.extra_files, p.gpg_key_id, p.base_url, p.sm2_enabled, p.enabled,
			p.created_at, p.updated_at,
			COALESCE((SELECT version FROM builds WHERE product_id = p.id AND status = 'success' ORDER BY created_at DESC LIMIT 1), ''),
			COALESCE((SELECT created_at FROM builds WHERE product_id = p.id ORDER BY created_at DESC LIMIT 1), '')
		FROM products p
		ORDER BY p.name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var items []models.ProductListItem
	for rows.Next() {
		var item models.ProductListItem
		var targetDistros, architectures string
		var productLines sql.NullString
		err := rows.Scan(
			&item.ID, &item.Name, &item.DisplayName, &item.Description,
			&item.SourceType, &item.SourceGithubOwner, &item.SourceGithubRepo, &item.SourceURLTemplate,
			&item.NfpmConfig, &targetDistros, &architectures, &productLines,
			&item.Maintainer, &item.Vendor, &item.Homepage, &item.License,
			&item.ScriptPostinstall, &item.ScriptPreremove,
			&item.SystemdService, &item.DefaultConfig, &item.DefaultConfigPath,
			&item.ExtraFiles, &item.GPGKeyID, &item.BaseURL, &item.SM2Enabled, &item.Enabled,
			&item.CreatedAt, &item.UpdatedAt,
			&item.LatestVersion, &item.LastBuildAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		item.TargetDistros = models.ParseJSONStringArray(targetDistros)
		item.Architectures = models.ParseJSONStringArray(architectures)
		if productLines.Valid {
			item.ProductLines = productLines.String
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ProductRepo) Update(p *models.Product) error {
	_, err := r.db.Exec(`
		UPDATE products SET
			name = ?, display_name = ?, description = ?,
			source_type = ?, source_github_owner = ?, source_github_repo = ?, source_url_template = ?,
			nfpm_config = ?, target_distros = ?, architectures = ?, product_lines = ?,
			maintainer = ?, vendor = ?, homepage = ?, license = ?,
			script_postinstall = ?, script_preremove = ?,
			systemd_service = ?, default_config = ?, default_config_path = ?,
			extra_files = ?, gpg_key_id = ?, base_url = ?, sm2_enabled = ?, enabled = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		p.Name, p.DisplayName, p.Description,
		p.SourceType, p.SourceGithubOwner, p.SourceGithubRepo, p.SourceURLTemplate,
		p.NfpmConfig, p.TargetDistrosJSON(), p.ArchitecturesJSON(), nilIfEmpty(p.ProductLines),
		p.Maintainer, p.Vendor, p.Homepage, p.License,
		p.ScriptPostinstall, p.ScriptPreremove,
		p.SystemdService, p.DefaultConfig, p.DefaultConfigPath,
		p.ExtraFiles, p.GPGKeyID, p.BaseURL, p.SM2Enabled, p.Enabled,
		p.ID,
	)
	return err
}

func (r *ProductRepo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM products WHERE id = ?", id)
	return err
}

func (r *ProductRepo) Duplicate(id int64) (int64, error) {
	src, err := r.GetByID(id)
	if err != nil {
		return 0, err
	}
	src.Name = src.Name + "-copy"
	src.DisplayName = src.DisplayName + " (Copy)"
	return r.Create(src)
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
