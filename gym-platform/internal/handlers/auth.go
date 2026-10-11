package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
	"github.com/jackc/pgx/v5"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/views/pages"
)

type Auth struct {
	q        *db.Queries
	sessions *scs.SessionManager
}

func NewAuth(q *db.Queries, sessions *scs.SessionManager) *Auth {
	return &Auth{q: q, sessions: sessions}
}

func (h *Auth) LoginPage(w http.ResponseWriter, r *http.Request) {
	if h.sessions.GetInt64(r.Context(), auth.SessionUserKey) != 0 {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	render(w, r, http.StatusOK, pages.Login(pages.LoginProps{
			Notice: h.sessions.PopString(r.Context(), "flash"),
		}))
}

func (h *Auth) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.PostFormValue("email")))
	password := r.PostFormValue("password")

	fail := func(msg string) {
		render(w, r, http.StatusUnprocessableEntity,
			pages.Login(pages.LoginProps{Email: email, Error: msg}))
	}

	if email == "" || password == "" {
		fail("Enter your email and password.")
		return
	}

	user, err := h.q.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			auth.BurnTime(password)
			fail("Invalid email or password.")
			return
		}
		log.Printf("login lookup: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	ok, err := auth.CheckPassword(password, user.PasswordHash)
	if err != nil || !ok {
		fail("Invalid email or password.")
		return
	}
	// Only revealed after the password is right, so it can't be used to
	// discover which emails belong to staff.
	if !user.IsActive {
		fail("This account has been deactivated. Contact the owner.")
		return
	}

	// New session token on login prevents session fixation.
	if err := h.sessions.RenewToken(r.Context()); err != nil {
		log.Printf("renew token: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	h.sessions.Put(r.Context(), auth.SessionUserKey, int(user.TokenVersion))
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)

}

func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.sessions.Destroy(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}