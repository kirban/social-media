package api

import (
	"encoding/json"
	"net/http"
)

func (h *Handlers) PostLogin(w http.ResponseWriter, r *http.Request) {
	var body PostLoginJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// TODO: implement login logic using h.Db, h.Logger, etc.
	w.WriteHeader(http.StatusOK)
}
