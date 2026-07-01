package api

import (
	"encoding/json"
	"net/http"

	"github.com/kirban/social-media/internal/middleware"
	"github.com/kirban/social-media/internal/model"
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

// // (PUT /post/delete/{id})
// func (h *Handlers) PutPostDeleteId(w http.ResponseWriter, r *http.Request, id PostId) {

// }

// // (GET /post/get/{id})
// func (h *Handlers) GetPostGetId(w http.ResponseWriter, r *http.Request, id PostId) {

// }

// // (PUT /post/update)
// func (h *Handlers) PutPostUpdate(w http.ResponseWriter, r *http.Request) {

// }
