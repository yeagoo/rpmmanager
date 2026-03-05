package auth

import (
	"crypto/subtle"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/ivmm/rpmmanager/internal/config"
)

type Service struct {
	cfg *config.AuthConfig
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

func NewService(cfg *config.AuthConfig) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) ValidatePassword(username, password string) error {
	if username != s.cfg.Username {
		return fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(s.cfg.PasswordHash), []byte(password)); err != nil {
		return fmt.Errorf("invalid credentials")
	}
	return nil
}

func (s *Service) GenerateJWT(username string) (string, error) {
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *Service) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (s *Service) ValidateAPIToken(token string) bool {
	if s.cfg.APIToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.APIToken)) == 1
}

func (s *Service) GetAPIToken() string {
	return s.cfg.APIToken
}
