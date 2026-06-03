package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handlers) GetUserById(w http.ResponseWriter, r *http.Request, id UserId) {
	if _, err := uuid.Parse(id); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.Logger.Error().Err(err).Msg("GetUserById: find user")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

func (h *Handlers) PostUserRegister(w http.ResponseWriter, r *http.Request) {
	var body PostUserRegisterJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if body.FirstName == nil || body.SecondName == nil || body.Password == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(*body.Password), bcrypt.DefaultCost)
	if err != nil {
		h.Logger.Error().Err(err).Msg("PostUserRegister: hash password")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	var birthdate *string
	if body.Birthdate != nil {
		s := body.Birthdate.Time.Format("2006-01-02")
		birthdate = &s
	}

	u := model.User{
		FirstName:    *body.FirstName,
		SecondName:   *body.SecondName,
		Birthdate:    birthdate,
		Biography:    strOrEmpty(body.Biography),
		City:         strOrEmpty(body.City),
		PasswordHash: string(hash),
	}

	id, err := h.UserRepo.Create(r.Context(), u)
	if err != nil {
		h.Logger.Error().Err(err).Msg("PostUserRegister: create user")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"user_id": id})
}

func (h *Handlers) GetUserSearch(w http.ResponseWriter, r *http.Request, params GetUserSearchParams) {
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
