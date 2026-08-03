package api

import (
	"net/http"
)

// (GET /friend/list)
func (h *Handlers) GetFriendList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	currentUserID, ok := h.currentUser(w, r, "GetFriendList")
	if !ok {
		return
	}

	friends, err := h.FriendsSvc.ListFriends(ctx, currentUserID)
	if err != nil {
		h.Logger.Error().Err(err).Msg("GetFriendList: list friends")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, friends)
}

// (PUT /friend/set/{user_id})
func (h *Handlers) PutFriendSetUserId(w http.ResponseWriter, r *http.Request, userId UserId) {
	if !parseUUID(w, r, userId) {
		return
	}

	ctx := r.Context()
	currentUserID, ok := h.currentUser(w, r, "PutFriendSetUserId")
	if !ok {
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
	currentUserID, ok := h.currentUser(w, r, "PutFriendDeleteUserId")
	if !ok {
		return
	}

	if err := h.FriendsSvc.DeleteFriend(ctx, currentUserID, userId); err != nil {
		h.Logger.Error().Err(err).Msg("PutFriendDeleteUserId: delete friend")
		writeError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
