package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/ivmm/rpmmanager/internal/config"
	"github.com/ivmm/rpmmanager/internal/models"
	"github.com/ivmm/rpmmanager/internal/repository"
)

// gpgSafeString rejects characters that could inject GPG batch directives.
var gpgUnsafeChars = regexp.MustCompile(`[%\n\r\x00]`)

type GPGService struct {
	cfg  *config.Config
	repo *repository.GPGKeyRepo
}

func NewGPGService(cfg *config.Config, repo *repository.GPGKeyRepo) *GPGService {
	return &GPGService{cfg: cfg, repo: repo}
}

func (s *GPGService) List() ([]models.GPGKey, error) {
	return s.repo.List()
}

func (s *GPGService) GetByID(id int64) (*models.GPGKey, error) {
	return s.repo.GetByID(id)
}

func (s *GPGService) Delete(id int64) error {
	key, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	// Remove from GPG keyring (log error but proceed with DB delete)
	if err := exec.Command(s.cfg.Tools.GPGPath, "--homedir", s.cfg.GPG.HomeDir,
		"--batch", "--yes", "--delete-secret-and-public-key", key.Fingerprint).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to delete key %s from keyring: %v\n", key.Fingerprint, err)
	}
	return s.repo.Delete(id)
}

func (s *GPGService) SetDefault(id int64) error {
	return s.repo.SetDefault(id)
}

func (s *GPGService) Export(id int64) (string, error) {
	key, err := s.repo.GetByID(id)
	if err != nil {
		return "", err
	}
	if key.PublicKeyArmor != "" {
		return key.PublicKeyArmor, nil
	}
	out, err := exec.Command(s.cfg.Tools.GPGPath, "--homedir", s.cfg.GPG.HomeDir,
		"--export", "--armor", key.Fingerprint).Output()
	if err != nil {
		return "", fmt.Errorf("gpg export: %w", err)
	}
	return string(out), nil
}

func (s *GPGService) ImportKey(keyData []byte) (*models.GPGKey, error) {
	// Write key to temp file with unique name
	tmpFile := filepath.Join(s.cfg.Storage.TempDir, fmt.Sprintf("import-key-%s.asc", uuid.New().String()[:8]))
	if err := os.WriteFile(tmpFile, keyData, 0600); err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile)

	// Import into GPG keyring
	cmd := exec.Command(s.cfg.Tools.GPGPath, "--homedir", s.cfg.GPG.HomeDir,
		"--batch", "--import", tmpFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg import: %s", stderr.String())
	}

	// Parse imported key info
	fingerprint := s.parseImportedFingerprint(stderr.String())
	if fingerprint == "" {
		return nil, fmt.Errorf("could not determine fingerprint from import output")
	}

	return s.syncKeyFromKeyring(fingerprint, true)
}

// validateGPGInput rejects strings containing characters that could inject GPG batch directives.
func validateGPGInput(name, value string) error {
	if gpgUnsafeChars.MatchString(value) {
		return fmt.Errorf("%s contains invalid characters (newlines, %% signs, or null bytes are not allowed)", name)
	}
	return nil
}

func (s *GPGService) GenerateKey(req *models.GenerateKeyRequest) (*models.GPGKey, error) {
	// Validate inputs to prevent GPG batch directive injection
	if err := validateGPGInput("name", req.Name); err != nil {
		return nil, err
	}
	if err := validateGPGInput("email", req.Email); err != nil {
		return nil, err
	}

	algo := req.Algorithm
	if algo == "" {
		algo = "RSA"
	}
	keyLength := req.KeyLength
	if keyLength == 0 {
		keyLength = 4096
	}
	expire := req.Expire
	if expire == "" {
		expire = "0"
	}
	if err := validateGPGInput("expire", expire); err != nil {
		return nil, err
	}

	// Build batch parameter file
	var paramContent string
	if strings.EqualFold(algo, "EDDSA") || strings.EqualFold(algo, "EdDSA") {
		paramContent = fmt.Sprintf(`%%no-protection
Key-Type: EDDSA
Key-Curve: ed25519
Subkey-Type: ECDH
Subkey-Curve: cv25519
Name-Real: %s
Name-Email: %s
Expire-Date: %s
%%commit
`, req.Name, req.Email, expire)
	} else {
		paramContent = fmt.Sprintf(`%%no-protection
Key-Type: RSA
Key-Length: %d
Subkey-Type: RSA
Subkey-Length: %d
Name-Real: %s
Name-Email: %s
Expire-Date: %s
%%commit
`, keyLength, keyLength, req.Name, req.Email, expire)
	}

	// Use unique temp file name
	paramFile := filepath.Join(s.cfg.Storage.TempDir, fmt.Sprintf("gen-key-params-%s", uuid.New().String()[:8]))
	if err := os.WriteFile(paramFile, []byte(paramContent), 0600); err != nil {
		return nil, err
	}
	defer os.Remove(paramFile)

	cmd := exec.Command(s.cfg.Tools.GPGPath, "--homedir", s.cfg.GPG.HomeDir,
		"--batch", "--gen-key", paramFile)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gpg gen-key: %s", stderr.String())
	}

	fingerprint := s.parseGeneratedFingerprint(stderr.String())
	if fingerprint == "" {
		return nil, fmt.Errorf("could not determine fingerprint from gen-key output")
	}

	return s.syncKeyFromKeyring(fingerprint, false)
}

func (s *GPGService) syncKeyFromKeyring(fingerprint string, imported bool) (*models.GPGKey, error) {
	// Get key details from keyring
	out, err := exec.Command(s.cfg.Tools.GPGPath, "--homedir", s.cfg.GPG.HomeDir,
		"--with-colons", "--list-keys", fingerprint).Output()
	if err != nil {
		return nil, fmt.Errorf("gpg list-keys: %w", err)
	}

	key := &models.GPGKey{
		Fingerprint: fingerprint,
		HasPrivate:  true,
	}

	// Parse colon output
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 10 {
			continue
		}
		switch fields[0] {
		case "pub":
			key.KeyLength, _ = parseInt(fields[2])
			key.Algorithm = mapAlgorithm(fields[3])
			key.KeyID = fields[4]
		case "uid":
			uid := fields[9]
			key.UIDName, key.UIDEmail = parseUID(uid)
		case "fpr":
			if key.Fingerprint == "" {
				key.Fingerprint = fields[9]
			}
		}
	}

	// Export public key
	pubKey, _ := exec.Command(s.cfg.Tools.GPGPath, "--homedir", s.cfg.GPG.HomeDir,
		"--export", "--armor", fingerprint).Output()
	key.PublicKeyArmor = string(pubKey)

	if key.Name == "" {
		key.Name = key.UIDName
	}
	if imported {
		key.Name = "Imported: " + key.UIDName
	}

	// Check if already exists in DB
	existing, _ := s.repo.GetByFingerprint(fingerprint)
	if existing != nil {
		return existing, nil
	}

	id, err := s.repo.Create(key)
	if err != nil {
		return nil, err
	}
	key.ID = id
	return key, nil
}

func (s *GPGService) parseImportedFingerprint(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "key") && strings.Contains(line, "imported") {
			// Extract key ID from "gpg: key ABCD1234: public key imported"
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "key" && i+1 < len(parts) {
					return strings.TrimSuffix(parts[i+1], ":")
				}
			}
		}
	}
	return ""
}

func (s *GPGService) parseGeneratedFingerprint(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "marked as ultimately trusted") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "key" && i+1 < len(parts) {
					return strings.TrimSuffix(parts[i+1], ":")
				}
			}
		}
	}
	return ""
}

func parseUID(uid string) (name, email string) {
	if idx := strings.Index(uid, " <"); idx >= 0 {
		name = uid[:idx]
		email = strings.TrimSuffix(uid[idx+2:], ">")
	} else {
		name = uid
	}
	return
}

func mapAlgorithm(code string) string {
	switch code {
	case "1":
		return "RSA"
	case "22":
		return "EdDSA"
	default:
		return code
	}
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
