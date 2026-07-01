package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/kirban/social-media/internal/middleware"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/service"
)

// (GET /post/feed)
func (h *Handlers) GetPostFeed(w http.ResponseWriter, r *http.Request, params GetPostFeedParams) {

}

// (POST /post/create)
func (h *Handlers) PostPostCreate(w http.ResponseWriter, r *http.Request) {
	var body PostPostCreateJSONBody

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to decode body")
		return
	}

	if body.Text == "" {
		writeError(w, r, http.StatusBadRequest, "text is not set")
		return
	}

	ctx := r.Context()
	authorUserID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		h.Logger.Error().Msg("PostPostCreate: failed to parse UserIDKey")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	dto := &model.Post{
		Text:      body.Text,
		CreatorID: authorUserID,
	}
	id, err := h.PostSvc.Create(ctx, dto)
	if err != nil {
		h.Logger.Error().Err(err).Msg("PostPostCreate: create post")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"post_id": id})

}

// (GET /post/get/{id})
func (h *Handlers) GetPostGetId(w http.ResponseWriter, r *http.Request, id PostId) {
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to parse id")
		return
	}

	post, err := h.PostSvc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		h.Logger.Error().Err(err).Msg("GetPostGetId: find post")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(post)
}

// (PUT /post/update)
func (h *Handlers) PutPostUpdate(w http.ResponseWriter, r *http.Request) {
	var body PutPostUpdateJSONBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to decode body")
		return
	}

	if _, err := uuid.Parse(body.Id); err != nil {
		writeError(w, r, http.StatusBadRequest, "failed to parse id")
		return
	}

	if body.Text == "" {
		writeError(w, r, http.StatusBadRequest, "text is not set")
		return
	}

	if err := h.PostSvc.Update(r.Context(), body.Id, &model.Post{Text: body.Text}); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		h.Logger.Error().Err(err).Msg("PutPostUpdate: update post")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// // (PUT /post/delete/{id})
// func (h *Handlers) PutPostDeleteId(w http.ResponseWriter, r *http.Request, id PostId) {

// }
