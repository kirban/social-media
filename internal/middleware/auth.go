package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kirban/social-media/internal/api"
)

type contextKey string

const UserIDKey contextKey = "userID"

type UserClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func Auth(secret string) func(http.Handler) http.Handler {
	key := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(api.BearerAuthScopes) == nil {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			var claims UserClaims
			token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return key, nil
			})
			if err != nil || !token.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
