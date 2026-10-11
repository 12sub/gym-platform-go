package handlers

import (
	"errors"
	"net/http"

	"github.com/alexedwards/scs/v2"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/permissions"
	"github.com/12sub/gym-platform/internal/services"
	"github.com/12sub/gym-platform/internal/views/pages"
)

type Accounts struct {
	svc      *services.Staff
	sessions *scs.SessionManager
}

func NewAccounts(svc *services.Staff, sessions *scs.SessionManager) *Accounts {
	return &Accounts{svc: svc, sessions: sessions}
}

// SetPasswordPage is public: it opens from a one-time link.
func (h *Accounts) SetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	info, err := h.svc.LookupToken(r.Context(), token)
	switch {
	case errors.Is(err, services.ErrTokenInvalid):
		render(w, r, http.StatusGone, pages.SetPassword(pages.SetPasswordProps{Error: err.Error()}))
		return
	case err != nil:
		serverError(w, "lookup token", err)
		return
	}
	render(w, r, http.StatusOK, pages.SetPassword(pages.SetPasswordProps{
		Token: token, Valid: true, Name: info.FullName, Purpose: info.Purpose,
	}))
}

func (h *Accounts) SetPasswordSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	info, err := h.svc.LookupToken(r.Context(), token)
	if errors.Is(err, services.ErrTokenInvalid) {
		render(w, r, http.StatusGone, pages.SetPassword(pages.SetPasswordProps{Error: err.Error()}))
		return
	}
	if err != nil {
		serverError(w, "lookup token", err)
		return
	}

	again := func(msg string) {
		render(w, r, http.StatusUnprocessableEntity, pages.SetPassword(pages.SetPasswordProps{
			Token: token, Valid: true, Name: info.FullName, Purpose: info.Purpose, Error: msg,
		}))
	}

	password := r.PostFormValue("password")
	if password != r.PostFormValue("confirm") {
		again("The two passwords don't match.")
		return
	}

	_, err = h.svc.CompleteSetup(r.Context(), token, password)
	switch {
	case errors.Is(err, services.ErrTokenInvalid):
		render(w, r, http.StatusGone, pages.SetPassword(pages.SetPasswordProps{Error: err.Error()}))
		return
	case services.IsUserError(err):
		again(err.Error())
		return
	case err != nil:
		serverError(w, "set password", err)
		return
	}

	h.sessions.Put(r.Context(), "flash", "Password saved. You can log in now.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Accounts) AccountPage(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	render(w, r, http.StatusOK, pages.AccountPage(
		newPage(r, "My account", "account"),
		pages.AccountView{
			Name:            u.FullName,
			Email:           u.Email,
			RoleLabel:       permissions.RoleLabel(u.RoleSlug),
			RoleDescription: permissions.RoleDescription(u.RoleSlug),
		},
		pages.PasswordFormData{},
	))
}

func (h *Accounts) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	u, _ := auth.UserFromContext(r.Context())

	f := pages.PasswordFormData{Errors: map[string]string{}}
	current, next := r.PostFormValue("current"), r.PostFormValue("new")
	if next != r.PostFormValue("confirm") {
		f.Errors["confirm"] = "The two passwords don't match."
		render(w, r, http.StatusUnprocessableEntity, pages.AccountPasswordForm(f))
		return
	}

	version, err := h.svc.ChangePassword(r.Context(), u.ID, u.Email, current, next)
	switch {
	case errors.Is(err, services.ErrWrongPassword):
		f.Errors["current"] = err.Error()
	case services.IsUserError(err):
		f.Errors["new"] = err.Error()
	case err != nil:
		serverError(w, "change password", err)
		return
	}
	if len(f.Errors) > 0 {
		render(w, r, http.StatusUnprocessableEntity, pages.AccountPasswordForm(f))
		return
	}

	// Every other session is now signed out. Keep this one alive on the new version.
	if err := h.sessions.RenewToken(r.Context()); err != nil {
		serverError(w, "renew session", err)
		return
	}
	h.sessions.Put(r.Context(), auth.SessionUserKey, u.ID)
	h.sessions.Put(r.Context(), auth.SessionVersionKey, int(version))

	render(w, r, http.StatusOK, pages.AccountPasswordForm(pages.PasswordFormData{Success: true}))
}