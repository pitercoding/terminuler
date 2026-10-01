package auth

import (
	"net/http"

	clerk "github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

// Middleware verifies the Clerk session token in the Authorization header.
// A token that cannot be verified, for example one issued by another Clerk
// instance, is rejected through writeError, so the 401 has the same JSON
// body as every other error instead of Clerk's default empty body.
func Middleware(writeError ErrorWriter, next http.Handler) http.Handler {
	return clerkhttp.WithHeaderAuthorization(
		clerkhttp.AuthorizationFailureHandler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeError(w, http.StatusUnauthorized, "unauthorized")
			}),
		),
	)(next)
}

func UserID(r *http.Request) (string, bool) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		return "", false
	}

	return claims.Subject, true
}
