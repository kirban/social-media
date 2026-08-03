package api

import (
	"net/http"
)

// (GET /dialog/list)
func (h *Handlers) GetDialogList(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.currentUser(w, r, "GetDialogList")
	if !ok {
		return
	}

	dialogs, err := h.DialogSvc.ListDialogs(r.Context(), userID)
	if err != nil {
		h.Logger.Error().Err(err).Msg("GetDialogList: failed to list dialogs")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, dialogs)
}

// (GET /dialog/{user_id}/list)
func (h *Handlers) GetDialogMessages(w http.ResponseWriter, r *http.Request, dstUser UserId, params GetDialogMessagesParams) {
	if !parseUUID(w, r, dstUser) {
		return
	}

	limit, offset := pageParams(params.Limit, params.Offset)

	ctx := r.Context()
	userID, ok := h.currentUser(w, r, "GetDialogMessages")
	if !ok {
		return
	}

	messages, err := h.DialogSvc.GetMessages(ctx, userID, dstUser, limit, offset)
	if err != nil {
		h.Logger.Error().Err(err).Msg("GetDialogMessages: failed to get dialog messages")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"messages": messages,
		"limit":    limit,
		"offset":   offset,
	})
}

// (POST /dialog/{user_id}/send)
func (h *Handlers) PostDialogUserIdSend(w http.ResponseWriter, r *http.Request, dstUser UserId) {
	if !parseUUID(w, r, dstUser) {
		return
	}

	ctx := r.Context()
	userID, ok := h.currentUser(w, r, "PostDialogUserIdSend")
	if !ok {
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
