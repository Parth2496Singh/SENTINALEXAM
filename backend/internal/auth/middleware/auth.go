// Package middleware provides HTTP middleware for authentication and authorization.
package middleware

import (
	"context"
	"net/http"

	"github.com/sentinelexam/backend/internal/assessment/service"
	"github.com/sentinelexam/backend/internal/auth/keycloak"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

// AuthMiddleware validates the JWT on every request and injects claims into ctx.
func AuthMiddleware(auth *keycloak.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := auth.ParseClaims(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"invalid or missing token"}}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromCtx retrieves auth claims from the request context.
func ClaimsFromCtx(ctx context.Context) *service.AuthClaims {
	if v, ok := ctx.Value(claimsKey).(*service.AuthClaims); ok {
		return v
	}
	return nil
}

// RequireRole returns a 403 if the authenticated user does not have all required roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromCtx(r.Context())
			if claims == nil {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"not authenticated"}}`, http.StatusUnauthorized)
				return
			}
			for _, role := range roles {
				if !claims.HasRole(role) {
					http.Error(w, `{"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`, http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole returns 403 if the user has NONE of the listed roles.
func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromCtx(r.Context())
			if claims == nil {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"not authenticated"}}`, http.StatusUnauthorized)
				return
			}
			for _, role := range roles {
				if claims.HasRole(role) {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`, http.StatusForbidden)
		})
	}
}
