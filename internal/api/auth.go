package api

import (
	"errors"
	"net/http"

	"github.com/kirban/social-media/internal/service"
)

func (h *Handlers) PostLogin(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody[PostLoginJSONRequestBody](w, r)
	if !ok {
		return
	}

	if body.Id == nil || body.Password == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !parseUUID(w, r, *body.Id) {
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

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}
