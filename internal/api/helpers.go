package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/kirban/social-media/internal/middleware"
)

func decodeBody[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to decode body")
		return v, false
	}
	return v, true
}

func parseUUID(w http.ResponseWriter, r *http.Request, id string) bool {
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to parse id")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	buf, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf)
}

// currentUser returns the authenticated user ID that the auth middleware placed
// in the context. A miss means a protected route was mounted without that
// middleware — a wiring bug, not a client error — so it reports 500.
func (h *Handlers) currentUser(w http.ResponseWriter, r *http.Request, op string) (string, bool) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		h.Logger.Error().Msgf("%s: failed to parse UserIDKey", op)
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return "", false
	}
	return userID, true
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// pageParams resolves the generated limit/offset query params, falling back to
// the defaults when absent or out of range. They arrive as *float32 because the
// spec types them as `number`; every paginated handler shares this conversion.
func pageParams(limit *Limit, offset *Offset) (int64, int64) {
	l, o := int64(DefaultLimit), int64(DefaultOffset)
	if limit != nil && *limit > 0 {
		l = int64(*limit)
	}
	if offset != nil && *offset >= 0 {
		o = int64(*offset)
	}
	return l, o
}
