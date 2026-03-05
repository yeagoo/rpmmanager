package repository

import (
	"database/sql"
	"fmt"

	"github.com/ivmm/rpmmanager/internal/models"
)

type GPGKeyRepo struct {
	db *sql.DB
}

func NewGPGKeyRepo(db *sql.DB) *GPGKeyRepo {
	return &GPGKeyRepo{db: db}
}

func (r *GPGKeyRepo) Create(k *models.GPGKey) (int64, error) {
	result, err := r.db.Exec(`
		INSERT INTO gpg_keys (name, fingerprint, key_id, uid_name, uid_email, algorithm, key_length,
			created_date, expires_date, has_private, public_key_armor, is_default)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		k.Name, k.Fingerprint, k.KeyID, k.UIDName, k.UIDEmail, k.Algorithm, k.KeyLength,
		k.CreatedDate, k.ExpiresDate, k.HasPrivate, k.PublicKeyArmor, k.IsDefault,
	)
	if err != nil {
		return 0, fmt.Errorf("insert gpg key: %w", err)
	}
	return result.LastInsertId()
}

func (r *GPGKeyRepo) GetByID(id int64) (*models.GPGKey, error) {
	k := &models.GPGKey{}
	var createdDate, expiresDate sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, name, fingerprint, key_id, uid_name, uid_email, algorithm, key_length,
			created_date, expires_date, has_private, public_key_armor, is_default, created_at, updated_at
		FROM gpg_keys WHERE id = ?`, id).Scan(
		&k.ID, &k.Name, &k.Fingerprint, &k.KeyID, &k.UIDName, &k.UIDEmail, &k.Algorithm, &k.KeyLength,
		&createdDate, &expiresDate, &k.HasPrivate, &k.PublicKeyArmor, &k.IsDefault, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if createdDate.Valid {
		k.CreatedDate = &createdDate.Time
	}
	if expiresDate.Valid {
		k.ExpiresDate = &expiresDate.Time
	}
	return k, nil
}

func (r *GPGKeyRepo) List() ([]models.GPGKey, error) {
	rows, err := r.db.Query(`
		SELECT id, name, fingerprint, key_id, uid_name, uid_email, algorithm, key_length,
			created_date, expires_date, has_private, public_key_armor, is_default, created_at, updated_at
		FROM gpg_keys ORDER BY is_default DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []models.GPGKey
	for rows.Next() {
		var k models.GPGKey
		var createdDate, expiresDate sql.NullTime
		if err := rows.Scan(
			&k.ID, &k.Name, &k.Fingerprint, &k.KeyID, &k.UIDName, &k.UIDEmail, &k.Algorithm, &k.KeyLength,
			&createdDate, &expiresDate, &k.HasPrivate, &k.PublicKeyArmor, &k.IsDefault, &k.CreatedAt, &k.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if createdDate.Valid {
			k.CreatedDate = &createdDate.Time
		}
		if expiresDate.Valid {
			k.ExpiresDate = &expiresDate.Time
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gpg keys: %w", err)
	}
	return keys, nil
}

func (r *GPGKeyRepo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM gpg_keys WHERE id = ?", id)
	return err
}

func (r *GPGKeyRepo) SetDefault(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("UPDATE gpg_keys SET is_default = FALSE"); err != nil {
		return fmt.Errorf("clear default: %w", err)
	}
	if _, err := tx.Exec("UPDATE gpg_keys SET is_default = TRUE WHERE id = ?", id); err != nil {
		return fmt.Errorf("set default: %w", err)
	}
	return tx.Commit()
}

func (r *GPGKeyRepo) GetByFingerprint(fp string) (*models.GPGKey, error) {
	k := &models.GPGKey{}
	var createdDate, expiresDate sql.NullTime
	err := r.db.QueryRow(`
		SELECT id, name, fingerprint, key_id, uid_name, uid_email, algorithm, key_length,
			created_date, expires_date, has_private, public_key_armor, is_default, created_at, updated_at
		FROM gpg_keys WHERE fingerprint = ?`, fp).Scan(
		&k.ID, &k.Name, &k.Fingerprint, &k.KeyID, &k.UIDName, &k.UIDEmail, &k.Algorithm, &k.KeyLength,
		&createdDate, &expiresDate, &k.HasPrivate, &k.PublicKeyArmor, &k.IsDefault, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if createdDate.Valid {
		k.CreatedDate = &createdDate.Time
	}
	if expiresDate.Valid {
		k.ExpiresDate = &expiresDate.Time
	}
	return k, nil
}
