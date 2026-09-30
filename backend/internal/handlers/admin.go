package handlers

import (
	"net/http"

	"github.com/pitercoding/terminuler/internal/auth"
)

type adminSessionResponse struct {
	UserID string `json:"user_id"`
}

// AdminSessionHandler returns the authenticated admin, letting the admin
// panel confirm its session token is accepted by the API. It must be
// wrapped by the admin authorization middleware.
func AdminSessionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, adminSessionResponse{
		UserID: userID,
	})
}
