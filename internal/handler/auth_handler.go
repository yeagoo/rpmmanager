package handler

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/ivmm/rpmmanager/internal/auth"
)

type AuthHandler struct {
	authService  *auth.Service
	challengeSvc *auth.ChallengeService
	rateLimiter  *auth.RateLimiter
}

func NewAuthHandler(authService *auth.Service, challengeSvc *auth.ChallengeService, rateLimiter *auth.RateLimiter) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		challengeSvc: challengeSvc,
		rateLimiter:  rateLimiter,
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Altcha   string `json:"altcha"`
}

func (h *AuthHandler) Challenge(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	if waitSec, blocked := h.rateLimiter.Check(ip); blocked {
		writeRateLimited(w, waitSec, "too many failed attempts, try again later")
		return
	}

	challenge, err := h.challengeSvc.Generate()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate challenge"})
		return
	}

	writeJSON(w, http.StatusOK, challenge)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	// Check rate limit (single call handles both block and backoff)
	if waitSec, blocked := h.rateLimiter.Check(ip); blocked || waitSec > 0 {
		msg := fmt.Sprintf("please wait %d seconds before trying again", waitSec)
		if blocked {
			msg = "too many failed attempts, try again later"
		}
		writeRateLimited(w, waitSec, msg)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Verify PoW challenge
	if req.Altcha == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "proof-of-work challenge required"})
		return
	}

	valid, err := h.challengeSvc.Verify(req.Altcha)
	if err != nil || !valid {
		h.rateLimiter.RecordFailure(ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid proof-of-work"})
		return
	}

	// Validate credentials
	if err := h.authService.ValidatePassword(req.Username, req.Password); err != nil {
		h.rateLimiter.RecordFailure(ip)
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	token, err := h.authService.GenerateJWT(req.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	h.rateLimiter.Reset(ip)

	writeJSON(w, http.StatusOK, auth.LoginResponse{
		Token:    token,
		Username: req.Username,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	username := auth.UserFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{
		"username": username,
	})
}

func writeRateLimited(w http.ResponseWriter, waitSec int, msg string) {
	w.Header().Set("Retry-After", strconv.Itoa(waitSec))
	writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
		"error":       msg,
		"retry_after": waitSec,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// clientIP extracts the real client IP.
// Only trusts X-Real-IP when the direct connection comes from a loopback
// or private IP (i.e., a local reverse proxy like Nginx/Caddy).
// X-Forwarded-For is NOT used because it can be spoofed by the client.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	// Only trust X-Real-IP from local/private network peers
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
			return strings.TrimSpace(xri)
		}
	}

	return host
}
