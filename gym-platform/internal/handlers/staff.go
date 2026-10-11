package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/permissions"
	"github.com/12sub/gym-platform/internal/services"
	"github.com/12sub/gym-platform/internal/views/pages"
)

type Staff struct {
	q   *db.Queries
	svc *services.Staff
}

func NewStaff(q *db.Queries, svc *services.Staff) *Staff {
	return &Staff{q: q, svc: svc}
}

// baseURL is the address staff will open the setup link on. Set APP_URL in
// production (for example https://gym.example.com); in development it's
// worked out from the request.
func baseURL(r *http.Request) string {
	if v := strings.TrimRight(os.Getenv("APP_URL"), "/"); v != "" {
		return v
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (h *Staff) tableView(r *http.Request, errMsg string, oob bool) (pages.StaffTableView, error) {
	u, _ := auth.UserFromContext(r.Context())
	list, err := h.q.ListStaff(r.Context())
	if err != nil {
		return pages.StaffTableView{}, err
	}
	rows := make([]pages.StaffRow, 0, len(list))
	for _, s := range list {
		rows = append(rows, pages.StaffRow{
			ID:        s.ID,
			Name:      s.FullName,
			Email:     s.Email,
			RoleSlug:  s.RoleSlug,
			RoleLabel: s.RoleName,
			Phone:     s.Phone,
			Position:  s.Position,
			IsActive:  s.IsActive,
			Pending:   s.PendingSetup,
			IsSelf:    s.ID == u.ID,
		})
	}
	return pages.StaffTableView{
		Rows:      rows,
		CanManage: can(r, permissions.StaffManage),
		Error:     errMsg,
		OOB:       oob,
	}, nil
}

func (h *Staff) Index(w http.ResponseWriter, r *http.Request) {
	v, err := h.tableView(r, "", false)
	if err != nil {
		serverError(w, "list staff", err)
		return
	}
	render(w, r, http.StatusOK, pages.StaffPage(newPage(r, "Staff & roles", "staff"), v))
}

func (h *Staff) NewForm(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, pages.StaffForm(pages.StaffFormData{
		Role:  "front_desk",
		Roles: permissions.Roles,
	}))
}

func (h *Staff) EditForm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s, err := h.q.GetStaff(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "get staff", err)
		return
	}
	render(w, r, http.StatusOK, pages.StaffForm(pages.StaffFormData{
		ID:       s.ID,
		FullName: s.FullName,
		Email:    s.Email,
		Role:     s.RoleSlug,
		Phone:    s.Phone,
		Position: s.Position,
		Roles:    permissions.Roles,
	}))
}

func readStaffForm(r *http.Request, id int64) pages.StaffFormData {
	v := func(k string) string { return strings.TrimSpace(r.PostFormValue(k)) }
	f := pages.StaffFormData{
		ID:       id,
		FullName: v("full_name"),
		Email:    strings.ToLower(v("email")),
		Role:     v("role"),
		Phone:    v("phone"),
		Position: v("position"),
		Roles:    permissions.Roles,
		Errors:   map[string]string{},
	}

	if f.FullName == "" {
		f.Errors["full_name"] = "Enter their full name."
	} else if len(f.FullName) > 100 {
		f.Errors["full_name"] = "Keep the name under 100 characters."
	}
	if id == 0 {
		if a, err := mail.ParseAddress(f.Email); err != nil || a.Address != f.Email {
			f.Errors["email"] = "Enter a valid email address."
		}
	}
	if !permissions.IsValidRole(f.Role) {
		f.Errors["role"] = "Choose a role."
	}
	if f.Phone != "" && !validPhone(f.Phone) {
		f.Errors["phone"] = "Enter a valid phone number."
	}
	if len(f.Position) > 60 {
		f.Errors["position"] = "Keep the job title under 60 characters."
	}
	return f
}

// staffFormError puts a service error on the right field of the form.
func staffFormError(f *pages.StaffFormData, err error) {
	switch {
	case errors.Is(err, services.ErrEmailTaken):
		f.Errors["email"] = err.Error()
	case errors.Is(err, services.ErrBadRole):
		f.Errors["role"] = err.Error()
	default:
		f.Errors["_"] = err.Error()
	}
}

func (h *Staff) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	f := readStaffForm(r, 0)
	if len(f.Errors) > 0 {
		render(w, r, http.StatusUnprocessableEntity, pages.StaffForm(f))
		return
	}

	u, _ := auth.UserFromContext(r.Context())
	token, _, err := h.svc.Create(r.Context(), u.ID, services.StaffInput{
		FullName: f.FullName, Email: f.Email, RoleSlug: f.Role, Phone: f.Phone, Position: f.Position,
	})
	switch {
	case services.IsUserError(err):
		staffFormError(&f, err)
		render(w, r, http.StatusUnprocessableEntity, pages.StaffForm(f))
		return
	case err != nil:
		serverError(w, "create staff", err)
		return
	}

	v, err := h.tableView(r, "", true)
	if err != nil {
		serverError(w, "list staff", err)
		return
	}
	render(w, r, http.StatusOK, pages.StaffCreated(pages.StaffLinkView{
		Title: "Send this link to " + f.FullName,
		Intro: "They open it, choose a password, and then sign in. Send it privately, for example by WhatsApp or SMS.",
		URL:   baseURL(r) + "/set-password/" + token,
		Hours: 72,
	}, v))
}

func (h *Staff) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	f := readStaffForm(r, id)
	if len(f.Errors) > 0 {
		render(w, r, http.StatusUnprocessableEntity, pages.StaffForm(f))
		return
	}

	u, _ := auth.UserFromContext(r.Context())
	err := h.svc.Update(r.Context(), u.ID, id, services.StaffInput{
		FullName: f.FullName, RoleSlug: f.Role, Phone: f.Phone, Position: f.Position,
	})
	switch {
	case errors.Is(err, services.ErrNotFound):
		http.NotFound(w, r)
		return
	case services.IsUserError(err):
		staffFormError(&f, err)
		render(w, r, http.StatusUnprocessableEntity, pages.StaffForm(f))
		return
	case err != nil:
		serverError(w, "update staff", err)
		return
	}

	v, err := h.tableView(r, "", false)
	if err != nil {
		serverError(w, "list staff", err)
		return
	}
	w.Header().Set("HX-Retarget", "#staff-table")
	w.Header().Set("HX-Reswap", "outerHTML")
	w.Header().Set("HX-Trigger", `{"close-dialog":"staff-dialog"}`)
	render(w, r, http.StatusOK, pages.StaffTable(v))
}

func (h *Staff) Toggle(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	cur, err := h.q.GetStaff(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "get staff", err)
		return
	}

	u, _ := auth.UserFromContext(r.Context())
	err = h.svc.SetActive(r.Context(), u.ID, id, !cur.IsActive)

	msg := ""
	switch {
	case errors.Is(err, services.ErrNotFound):
		http.NotFound(w, r)
		return
	case services.IsUserError(err):
		msg = err.Error()
	case err != nil:
		serverError(w, "toggle staff", err)
		return
	}

	v, err := h.tableView(r, msg, false)
	if err != nil {
		serverError(w, "list staff", err)
		return
	}
	status := http.StatusOK
	if msg != "" {
		status = http.StatusUnprocessableEntity
	}
	render(w, r, status, pages.StaffTable(v))
}

// ResetLink issues a fresh one-time link and shows it in the dialog.
func (h *Staff) ResetLink(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	cur, err := h.q.GetStaff(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "get staff", err)
		return
	}

	u, _ := auth.UserFromContext(r.Context())
	token, purpose, ttl, err := h.svc.IssueLink(r.Context(), u.ID, id)
	switch {
	case services.IsUserError(err):
		render(w, r, http.StatusOK, pages.StaffLink(pages.StaffLinkView{Title: "Couldn't create a link", Error: err.Error()}))
		return
	case err != nil:
		serverError(w, "issue link", err)
		return
	}

	title, intro := "Send this reset link to "+cur.FullName, "Opening it lets them choose a new password. Their old password stops working once they do, and they'll be signed out everywhere."
	if purpose == "invite" {
		title, intro = "Send this setup link to "+cur.FullName, "They open it, choose a password, and then sign in. Any earlier link no longer works."
	}
	render(w, r, http.StatusOK, pages.StaffLink(pages.StaffLinkView{
		Title: title,
		Intro: intro,
		URL:   baseURL(r) + "/set-password/" + token,
		Hours: int(ttl.Hours()),
	}))
}

func (h *Staff) Roles(w http.ResponseWriter, r *http.Request) {
	var m pages.RoleMatrix
	for _, ro := range permissions.Roles {
		m.Roles = append(m.Roles, pages.RoleCol{Label: ro.Label, Description: ro.Description})
	}
	for _, pi := range permissions.AllPermissions {
		if len(m.Groups) == 0 || m.Groups[len(m.Groups)-1].Name != pi.Group {
			m.Groups = append(m.Groups, pages.MatrixGroup{Name: pi.Group})
		}
		row := pages.MatrixRow{Label: pi.Label}
		for _, ro := range permissions.Roles {
			row.Allowed = append(row.Allowed, permissions.RoleHas(ro.Slug, pi.Permission))
		}
		g := &m.Groups[len(m.Groups)-1]
		g.Rows = append(g.Rows, row)
	}
	render(w, r, http.StatusOK, pages.RolesPage(newPage(r, "Staff & roles", "staff"), m))
}