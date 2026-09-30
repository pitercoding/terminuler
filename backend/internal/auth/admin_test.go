package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	clerk "github.com/clerk/clerk-sdk-go/v2"
)

// withSessionClaims simulates a request whose session token was already
// verified by the Clerk middleware for the given user ID.
func withSessionClaims(r *http.Request, userID string) *http.Request {
	claims := &clerk.SessionClaims{
		RegisteredClaims: clerk.RegisteredClaims{
			Subject: userID,
		},
	}

	return r.WithContext(
		clerk.ContextWithSessionClaims(r.Context(), claims),
	)
}

func TestRequireAdmin_AllowsAdmin(t *testing.T) {
	adminUserID := "user_admin"

	request := withSessionClaims(
		httptest.NewRequest(http.MethodGet, "/admin", nil),
		adminUserID,
	)

	nextCalled := false

	handler := RequireAdmin(
		adminUserID,
		func(w http.ResponseWriter, status int, message string) {
			t.Fatalf("writeError should not be called")
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestRequireAdmin_RejectsUnauthenticatedUser(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodGet,
		"/admin",
		nil,
	)

	handler := RequireAdmin(
		"user_admin",
		func(w http.ResponseWriter, status int, message string) {
			w.WriteHeader(status)
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		}),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			recorder.Code,
		)
	}
}

func TestRequireAdmin_RejectsNonAdminUser(t *testing.T) {
	request := withSessionClaims(
		httptest.NewRequest(http.MethodGet, "/admin", nil),
		"user_other",
	)

	handler := RequireAdmin(
		"user_admin",
		func(w http.ResponseWriter, status int, message string) {
			w.WriteHeader(status)
		},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		}),
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			recorder.Code,
		)
	}
}
