package api

import (
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/miguelscastro/ignite/go/05-gobid/internal/jsonutils"
)

func (api *Api) handleGetCSRFToken(w http.ResponseWriter, r *http.Request) {
	token := csrf.Token(r)
	jsonutils.EncodeJSON(w, r, http.StatusOK, map[string]any{
		"csrf_token": token,
	})
}

func (api *Api) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !api.Sessions.Exists(r.Context(), "AuthenticatedUserID") {
			jsonutils.EncodeJSON(w, r, http.StatusUnauthorized, map[string]any{
				"message": "user must be logged in",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
