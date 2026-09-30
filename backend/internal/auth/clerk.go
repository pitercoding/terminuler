package auth

import (
	"net/http"

	clerk "github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
)

func Middleware(next http.Handler) http.Handler {
	return clerkhttp.WithHeaderAuthorization()(next)
}

func UserID(r *http.Request) (string, bool) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		return "", false
	}

	return claims.Subject, true
}
