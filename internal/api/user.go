package api

import (
	"errors"
	"net/http"

	"github.com/kirban/social-media/internal/service"
)

func (h *Handlers) GetUserById(w http.ResponseWriter, r *http.Request, id UserId) {
	if !parseUUID(w, r, id) {
		return
	}

	user, err := h.UserSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		h.Logger.Error().Err(err).Msg("GetUserById: find user")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *Handlers) PostUserRegister(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody[PostUserRegisterJSONRequestBody](w, r)
	if !ok {
		return
	}

	if body.FirstName == nil || body.SecondName == nil || body.Password == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var birthdate *string
	if body.Birthdate != nil {
		s := body.Birthdate.Time.Format("2006-01-02")
		birthdate = &s
	}

	id, err := h.UserSvc.Register(r.Context(), *body.FirstName, *body.SecondName, *body.Password, birthdate, strOrEmpty(body.Biography), strOrEmpty(body.City))
	if err != nil {
		h.Logger.Error().Err(err).Msg("PostUserRegister: register user")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"user_id": id})
}

func (h *Handlers) GetUserSearch(w http.ResponseWriter, r *http.Request, params GetUserSearchParams) {
	users, err := h.UserSvc.SearchByNames(r.Context(), params.FirstName, params.LastName)
	if err != nil {
		h.Logger.Error().Err(err).Msg("GetUserSearch")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, users)
}
