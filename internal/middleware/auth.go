package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kirban/social-media/internal/model"
)

type contextKey string

const UserIDKey contextKey = "userID"

func Auth(secret string, securedRouteKey any) func(http.Handler) http.Handler {
	key := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Context().Value(securedRouteKey) == nil {
				next.ServeHTTP(w, r)
				return
			}

			tokenStr, ok := tokenFromRequest(r)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			var claims model.UserClaims
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

// tokenFromRequest pulls the bearer token from the Authorization header, falling
// back to a "token" query parameter on WebSocket upgrades only.
//
// The fallback exists because the browser WebSocket constructor cannot set
// request headers, so a browser client has no other way to authenticate the
// handshake. It is deliberately not accepted on ordinary requests: a token in a
// URL leaks into access logs, proxy logs, browser history and Referer headers,
// which is precisely what the header avoids.
func tokenFromRequest(r *http.Request) (string, bool) {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer "), true
	}

	if isWebSocketUpgrade(r) {
		if t := r.URL.Query().Get("token"); t != "" {
			return t, true
		}
	}

	return "", false
}

// isWebSocketUpgrade reports whether r is a WebSocket handshake. Connection is a
// comma-separated token list and both headers are case-insensitive, so neither
// can be compared with a plain string equality.
func isWebSocketUpgrade(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return false
	}

	for _, token := range strings.Split(r.Header.Get("Connection"), ",") {
		if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
			return true
		}
	}

	return false
}
