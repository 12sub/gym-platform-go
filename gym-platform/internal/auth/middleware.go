package auth

import (
	"context"
	"net/http"

	"github.com/alexedwards/scs/v2"

	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/permissions"
)

type User struct {
	ID       int64
	FullName string
	Email    string
	RoleSlug string
}

type userCtxKey struct{}

func UserFromContext(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(userCtxKey{}).(User)
	return u, ok
}

type Middleware struct {
	sessions *scs.SessionManager
	q        *db.Queries
}

func NewMiddleware(sessions *scs.SessionManager, q *db.Queries) *Middleware {
	return &Middleware{sessions: sessions, q: q}
}

// RequireAuth loads the current user from the database on every request, so a
// deactivated account loses access immediately instead of at session expiry.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := m.sessions.GetInt64(r.Context(), SessionUserKey)
		if id == 0 {
			redirectToLogin(w, r)
			return
		}

		row, err := m.q.GetUserByID(r.Context(), id)
		if err != nil || !row.IsActive {
			_ = m.sessions.Destroy(r.Context())
			redirectToLogin(w, r)
			return
		}

		u := User{ID: row.ID, FullName: row.FullName, Email: row.Email, RoleSlug: row.RoleSlug}
		ctx := context.WithValue(r.Context(), userCtxKey{}, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Protect = must be logged in AND the role must have the permission.
func (m *Middleware) Protect(p permissions.Permission, next http.Handler) http.Handler {
	return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := UserFromContext(r.Context())
		if !permissions.RoleHas(u.RoleSlug, p) {
			http.Error(w, "You don't have permission to view this page.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	// HTMX requests need a header-based redirect, otherwise the login page
	// would be swapped into a small part of the current page.
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
