package auth

import "net/http"

type ErrorWriter func(
	w http.ResponseWriter,
	status int,
	message string,
)

// AdminMiddleware verifies the Clerk session token in the Authorization
// header and then only lets the admin user through.
func AdminMiddleware(
	adminUserID string,
	writeError ErrorWriter,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return Middleware(
			writeError,
			RequireAdmin(adminUserID, writeError, next),
		)
	}
}

func RequireAdmin(
	adminUserID string,
	writeError ErrorWriter,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserID(r)

		if !ok {
			writeError(
				w,
				http.StatusUnauthorized,
				"unauthorized",
			)
			return
		}

		if userID != adminUserID {
			writeError(
				w,
				http.StatusForbidden,
				"forbidden",
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
