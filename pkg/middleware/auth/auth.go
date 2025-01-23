package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/ranson21/ranor-common/pkg/auth/claims"
	"github.com/ranson21/ranor-common/pkg/auth/token"
	"github.com/ranson21/ranor-common/pkg/logger"
	"go.uber.org/zap"
)

type contextKey string

const (
	ContextKeyClaims = contextKey("claims")
)

type Middleware interface {
	Authenticate(next http.Handler) http.Handler
	RequirePermissions(permissions ...claims.Permission) func(http.Handler) http.Handler
	RequireAnyPermission(permissions ...claims.Permission) func(http.Handler) http.Handler
}

type AuthMiddleware struct {
	validator token.TokenValidator
	logger    logger.Logger
}

func NewAuthMiddleware(v token.TokenValidator, l logger.Logger) Middleware {
	return &AuthMiddleware{
		validator: v,
		logger:    l,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			m.logger.Warn("No token provided in request")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := m.validator.ValidateToken(r.Context(), token)
		if err != nil {
			m.logger.Error("Token validation failed",
				zap.Error(err),
			)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Check if token is expired
		if claims.IsExpired() {
			m.logger.Warn("Expired token",
				zap.String("user_id", claims.UserID),
				zap.Time("expired_at", claims.ExpiryTime()),
			)
			http.Error(w, "Token expired", http.StatusUnauthorized)
			return
		}

		m.logger.Info("Successful authentication",
			zap.String("user_id", claims.UserID),
			zap.String("app_id", claims.AppID),
			zap.Duration("expires_in", claims.TimeUntilExpiry()),
		)

		ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequirePermissions(required ...claims.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ContextKeyClaims).(*claims.Claims)
			if !ok {
				m.logger.Error("No claims found in context")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !claims.HasAllPermissions(required...) {
				m.logger.Warn("Insufficient permissions",
					zap.String("user_id", claims.UserID),
					zap.Any("required_perms", required),
					zap.Strings("user_roles", claims.Roles),
				)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *AuthMiddleware) RequireAnyPermission(permissions ...claims.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(ContextKeyClaims).(*claims.Claims)
			if !ok {
				m.logger.Error("No claims found in context")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !claims.HasAnyPermission(permissions...) {
				m.logger.Warn("Insufficient permissions",
					zap.String("user_id", claims.UserID),
					zap.Any("required_perms", permissions),
					zap.Strings("user_roles", claims.Roles),
				)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) string {
	bearerToken := r.Header.Get("Authorization")
	parts := strings.Split(bearerToken, " ")
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}
	return ""
}
