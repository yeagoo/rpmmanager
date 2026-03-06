package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try API token first (X-API-Token header)
		if apiToken := r.Header.Get("X-API-Token"); apiToken != "" {
			if s.ValidateAPIToken(apiToken) {
				ctx := context.WithValue(r.Context(), userContextKey, s.cfg.Username)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try Bearer token (JWT) from header or query param
		authHeader := r.Header.Get("Authorization")
		tokenStr := ""
		if authHeader != "" {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
				return
			}
		} else if qToken := r.URL.Query().Get("token"); qToken != "" {
			// Fallback: ?token= query param (for WebSocket connections)
			tokenStr = qToken
		} else {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		claims, err := s.ValidateJWT(tokenStr)
		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) string {
	user, _ := ctx.Value(userContextKey).(string)
	return user
}
