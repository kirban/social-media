package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/kirban/social-media/internal/service"
)

func (h *Handlers) PostLogin(w http.ResponseWriter, r *http.Request) {
	var body PostLoginJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if body.Id == nil || body.Password == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(*body.Id); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := h.UserSvc.Login(r.Context(), *body.Id, *body.Password)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.Logger.Error().Err(err).Msg("PostLogin: login")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
