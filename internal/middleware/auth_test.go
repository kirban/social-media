package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kirban/social-media/internal/model"
)

const testSecret = "test-secret"

// scopeKey stands in for api.BearerAuthScopes: Auth treats a route as protected
// only when this key is present in the request context.
type scopeKeyType struct{}

var scopeKey = scopeKeyType{}

func signToken(t *testing.T, userID string, expiry time.Duration) string {
	t.Helper()

	claims := model.UserClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// serve runs a request through Auth on a protected route and reports the status
// plus the user ID the handler saw.
func serve(t *testing.T, r *http.Request) (int, string) {
	t.Helper()

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := r.Context().Value(UserIDKey).(string); ok {
			gotUserID = id
		}
		w.WriteHeader(http.StatusOK)
	})

	// Seed the scope key the way the generated router does, marking the route
	// as protected.
	ctx := context.WithValue(r.Context(), scopeKey, []string{})

	rec := httptest.NewRecorder()
	Auth(testSecret, scopeKey)(next).ServeHTTP(rec, r.WithContext(ctx))
	return rec.Code, gotUserID
}

func wsUpgradeRequest(target string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, target, nil)
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Connection", "Upgrade")
	return r
}

func TestAuthAcceptsBearerHeader(t *testing.T) {
	token := signToken(t, "user-1", time.Hour)
	r := httptest.NewRequest(http.MethodGet, "/post/feed", nil)
	r.Header.Set("Authorization", "Bearer "+token)

	status, userID := serve(t, r)

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if userID != "user-1" {
		t.Fatalf("userID = %q, want %q", userID, "user-1")
	}
}

func TestAuthAcceptsQueryTokenOnWebSocketUpgrade(t *testing.T) {
	token := signToken(t, "user-2", time.Hour)

	status, userID := serve(t, wsUpgradeRequest("/post/feed/posted?token="+token))

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if userID != "user-2" {
		t.Fatalf("userID = %q, want %q", userID, "user-2")
	}
}

// The query-parameter fallback must never apply to ordinary requests: a token in
// a URL leaks into logs and Referer headers.
func TestAuthRejectsQueryTokenOnNonUpgradeRequest(t *testing.T) {
	token := signToken(t, "user-3", time.Hour)
	r := httptest.NewRequest(http.MethodGet, "/post/feed?token="+token, nil)

	status, userID := serve(t, r)

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
	if userID != "" {
		t.Fatalf("userID = %q, want empty", userID)
	}
}

// A request carrying only the Upgrade header, without Connection: Upgrade, is
// not a valid handshake and must not unlock the fallback.
func TestAuthRejectsQueryTokenWithoutConnectionUpgrade(t *testing.T) {
	token := signToken(t, "user-4", time.Hour)
	r := httptest.NewRequest(http.MethodGet, "/post/feed/posted?token="+token, nil)
	r.Header.Set("Upgrade", "websocket")

	status, _ := serve(t, r)

	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestAuthRejectsExpiredAndInvalidTokens(t *testing.T) {
	cases := map[string]string{
		"expired":      signToken(t, "user-5", -time.Minute),
		"wrong secret": mustSignWith(t, "other-secret"),
		"malformed":    "not-a-jwt",
		"empty":        "",
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			r := wsUpgradeRequest("/post/feed/posted?token=" + token)

			status, userID := serve(t, r)

			if status != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", status)
			}
			if userID != "" {
				t.Fatalf("userID = %q, want empty", userID)
			}
		})
	}
}

func mustSignWith(t *testing.T, secret string) string {
	t.Helper()

	claims := model.UserClaims{
		UserID: "user-x",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// A route without the scope key is public and must pass through untouched.
func TestAuthSkipsUnprotectedRoutes(t *testing.T) {
	var called bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	Auth(testSecret, scopeKey)(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/user/search", nil))

	if !called {
		t.Fatal("handler was not called on a public route")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
