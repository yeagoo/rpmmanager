package models

import "time"

type GPGKey struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Fingerprint    string     `json:"fingerprint"`
	KeyID          string     `json:"key_id"`
	UIDName        string     `json:"uid_name"`
	UIDEmail       string     `json:"uid_email"`
	Algorithm      string     `json:"algorithm"`
	KeyLength      int        `json:"key_length"`
	CreatedDate    *time.Time `json:"created_date"`
	ExpiresDate    *time.Time `json:"expires_date"`
	HasPrivate     bool       `json:"has_private"`
	PublicKeyArmor string     `json:"public_key_armor"`
	IsDefault      bool       `json:"is_default"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type GenerateKeyRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Algorithm string `json:"algorithm"`  // RSA or EDDSA
	KeyLength int    `json:"key_length"` // 2048, 3072, 4096 for RSA
	Expire    string `json:"expire"`     // "0" = never, "1y", "2y", etc.
}
