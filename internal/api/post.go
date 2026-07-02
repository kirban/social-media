package api

import (
	"errors"
	"net/http"

	"github.com/kirban/social-media/internal/middleware"
	"github.com/kirban/social-media/internal/model"
	"github.com/kirban/social-media/internal/service"
)

const DefaultLimit = 10
const DefaultOffset = 0

// (GET /post/feed)
func (h *Handlers) GetPostFeed(w http.ResponseWriter, r *http.Request, params GetPostFeedParams) {
	var limit, offset int64

	if params.Limit == nil {
		limit = DefaultLimit
	} else {
		limit = int64(*params.Limit)
	}

	if params.Offset == nil {
		offset = DefaultOffset
	} else {
		offset = int64(*params.Offset)
	}

	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		h.Logger.Error().Msg("GetPostFeed: failed to parse UserIDKey")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	feed, err := h.PostSvc.GetFeed(ctx, userID, limit, offset)
	if err != nil {
		h.Logger.Error().Msg("GetPostFeed: failed to get feed for user")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"posts":  feed,
		"limit":  limit,
		"offset": offset,
	})

}

// (POST /post/create)
func (h *Handlers) PostPostCreate(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody[PostPostCreateJSONBody](w, r)
	if !ok {
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

	id, err := h.PostSvc.Create(ctx, &model.Post{Text: body.Text, CreatorID: authorUserID})
	if err != nil {
		h.Logger.Error().Err(err).Msg("PostPostCreate: create post")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"post_id": id})
}

// (GET /post/get/{id})
func (h *Handlers) GetPostGetId(w http.ResponseWriter, r *http.Request, id PostId) {
	if !parseUUID(w, r, id) {
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

	writeJSON(w, http.StatusOK, post)
}

// (PUT /post/update)
func (h *Handlers) PutPostUpdate(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody[PutPostUpdateJSONBody](w, r)
	if !ok {
		return
	}

	if !parseUUID(w, r, body.Id) {
		return
	}

	if body.Text == "" {
		writeError(w, r, http.StatusBadRequest, "text is not set")
		return
	}

	// todo: add check that user is creator of the post

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

// (PUT /post/delete/{id})
func (h *Handlers) PutPostDeleteId(w http.ResponseWriter, r *http.Request, id PostId) {
	if !parseUUID(w, r, id) {
		return
	}

	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		h.Logger.Error().Msg("PutPostDeleteId: failed to parse UserIDKey")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	// todo: add check that user is creator of the post or admin

	if err := h.PostSvc.Delete(ctx, id, userID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not found")
			return
		}
		h.Logger.Error().Err(err).Msg("PutPostDeleteId: delete post")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
