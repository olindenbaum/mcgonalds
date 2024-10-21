package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/olindenbaum/mcgonalds/internal/config"
	"github.com/olindenbaum/mcgonalds/internal/utils"
)

type contextKey string

const (
	ContextUserID   contextKey = "userID"
	ContextUsername contextKey = "username"
)

// AuthMiddleware validates JWT tokens and adds user info to the request context
func AuthMiddleware(jwtConfig *config.JWTConfig) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Expect the header to be in the format "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]
			claims, err := utils.ValidateJWT(tokenStr)
			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextUsername, claims.Username)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
