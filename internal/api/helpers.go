package api

import (
	"encoding/json"
	"net/http"

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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
