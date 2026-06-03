package api

import (
	"encoding/json"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	reqID := chimiddleware.GetReqID(r.Context())
	body := N5xx{Message: message}
	if reqID != "" {
		body.RequestId = &reqID
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body) //nolint:errcheck
}
