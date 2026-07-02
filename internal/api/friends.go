package api

import (
	"net/http"

	"github.com/kirban/social-media/internal/middleware"
)

// (PUT /friend/set/{user_id})
func (h *Handlers) PutFriendSetUserId(w http.ResponseWriter, r *http.Request, userId UserId) {
	if !parseUUID(w, r, userId) {
		return
	}

	ctx := r.Context()
	currentUserID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		h.Logger.Error().Msg("PutFriendSetUserId: failed to parse UserIDKey")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.FriendsSvc.AddFriend(ctx, currentUserID, userId); err != nil {
		h.Logger.Error().Err(err).Msg("PutFriendSetUserId: add friend")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// (PUT /friend/delete/{user_id})
func (h *Handlers) PutFriendDeleteUserId(w http.ResponseWriter, r *http.Request, userId UserId) {
	if !parseUUID(w, r, userId) {
		return
	}

	ctx := r.Context()
	currentUserID, ok := ctx.Value(middleware.UserIDKey).(string)
	if !ok {
		h.Logger.Error().Msg("PutFriendDeleteUserId: failed to parse UserIDKey")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.FriendsSvc.DeleteFriend(ctx, currentUserID, userId); err != nil {
		h.Logger.Error().Err(err).Msg("PutFriendDeleteUserId: delete friend")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
