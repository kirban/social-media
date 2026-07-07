package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
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

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parsePageParams(r *http.Request, defaultLimit, defaultOffset int64) (limit, offset int64) {
	limit = defaultLimit
	offset = defaultOffset
	q := r.URL.Query()
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}
