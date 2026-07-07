package api

import (
	"net/http"

	"github.com/kirban/social-media/internal/middleware"
)

// (GET /dialog/{user_id}/list)
func (h *Handlers) GetDialogUserIdList(w http.ResponseWriter, r *http.Request, dstUser UserId) {
	if !parseUUID(w, r, dstUser) {
		return
	}

	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		h.Logger.Error().Msg("GetDialogUserIdList: failed to parse UserIDKey")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	messages, err := h.DialogSvc.GetMessages(ctx, userID, dstUser)
	if err != nil {
		h.Logger.Error().Err(err).Msg("GetDialogUserIdList: failed to get dialog messages")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"messages": messages,
		// "limit":  limit,
		// "offset": offset,
	})
}

// (POST /dialog/{user_id}/send)
func (h *Handlers) PostDialogUserIdSend(w http.ResponseWriter, r *http.Request, dstUser UserId) {
	if !parseUUID(w, r, dstUser) {
		return
	}

	ctx := r.Context()
	userID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		h.Logger.Error().Msg("PostDialogUserIdSend: failed to parse UserIDKey")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	body, ok := decodeBody[PostDialogUserIdSendJSONBody](w, r)
	if !ok {
		return
	}

	if body.Text == "" {
		writeError(w, r, http.StatusBadRequest, "text is not set")
		return
	}

	mID, err := h.DialogSvc.SendMessage(ctx, userID, dstUser, body.Text)
	if err != nil {
		h.Logger.Error().Err(err).Msg("PostDialogUserIdSend: failed to send message")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id": *mID,
	})
}
