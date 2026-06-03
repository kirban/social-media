package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/repository"
	"golang.org/x/crypto/bcrypt"
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

	user, err := h.UserRepo.FindByID(r.Context(), *body.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.Logger.Error().Err(err).Msg("PostLogin: find user")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(*body.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	claims := model.UserClaims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.JWTSecret))
	if err != nil {
		h.Logger.Error().Err(err).Msg("PostLogin: sign token")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
