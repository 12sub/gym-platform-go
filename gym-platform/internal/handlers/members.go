package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/12sub/gym-platform/internal/clock"
	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/permissions"
	"github.com/12sub/gym-platform/internal/views/pages"
)

const memberPageSize = 20

const dateLayout = "2006-01-02"

type Members struct{ q *db.Queries }

func NewMembers(q *db.Queries) *Members { return &Members{q: q} }

func filterFromValues(v url.Values) pages.MembersFilter {
	f := pages.MembersFilter{
		Search: strings.TrimSpace(v.Get("q")),
		Status: v.Get("status"),
		Page:   1,
	}
	switch f.Status {
	case "active", "expired", "none":
	default:
		f.Status = "all"
	}
	if n, err := strconv.Atoi(v.Get("page")); err == nil && n > 1 {
		f.Page = n
	}
	return f
}

// currentFilter reads the filter from the page the browser is showing, so a
// list that refreshes after an edit keeps the user's search.
func currentFilter(r *http.Request) pages.MembersFilter {
	if u, err := url.Parse(r.Header.Get("HX-Current-URL")); err == nil {
		return filterFromValues(u.Query())
	}
	return filterFromValues(url.Values{})
}

func (h *Members) results(ctx context.Context, f pages.MembersFilter) ([]pages.MemberRow, bool, error) {
	// Ask for one extra row so we know whether a next page exists.
	list, err := h.q.ListMembers(ctx, db.ListMembersParams{
		Search:     f.Search,
		Filter:     f.Status,
		PageSize:   memberPageSize + 1,
		PageOffset: int32((f.Page - 1) * memberPageSize),
	})
	if err != nil {
		return nil, false, err
	}
	hasNext := len(list) > memberPageSize
	if hasNext {
		list = list[:memberPageSize]
	}
	rows := make([]pages.MemberRow, 0, len(list))
	for _, m := range list {
		rows = append(rows, pages.MemberRow{
			ID:       m.ID,
			Code:     m.MemberCode,
			Name:     m.FullName,
			Email:    deref(m.Email),
			Phone:    m.Phone,
			Plan:     m.PlanName,
			State:    m.SubState,
			EndsOn:   m.EndsOnLabel,
			IsActive: m.IsActive,
		})
	}
	return rows, hasNext, nil
}

func (h *Members) Index(w http.ResponseWriter, r *http.Request) {
	f := filterFromValues(r.URL.Query())
	rows, hasNext, err := h.results(r.Context(), f)
	if err != nil {
		serverError(w, "list members", err)
		return
	}
	canManage := can(r, permissions.MembersManage)

	if isHTMX(r) {
		render(w, r, http.StatusOK, pages.MembersResults(f, rows, hasNext, canManage))
		return
	}
	render(w, r, http.StatusOK,
		pages.MembersPage(newPage(r, "Members", "members"), f, rows, hasNext, canManage))
}

func (h *Members) NewForm(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, pages.MemberForm(pages.MemberFormData{
		JoinedOn: clock.Today().Format(dateLayout),
	}))
}

func (h *Members) EditForm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	m, err := h.q.GetMember(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "get member", err)
		return
	}
	f := pages.MemberFormData{
		ID:             m.ID,
		FullName:       m.FullName,
		Phone:          m.Phone,
		Email:          deref(m.Email),
		EmergencyName:  deref(m.EmergencyContactName),
		EmergencyPhone: deref(m.EmergencyContactPhone),
		JoinedOn:       m.JoinedOn.Format(dateLayout),
	}
	if m.DateOfBirth != nil {
		f.DateOfBirth = m.DateOfBirth.Format(dateLayout)
	}
	render(w, r, http.StatusOK, pages.MemberForm(f))
}

func (h *Members) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, 0) }

func (h *Members) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.save(w, r, id)
}

func (h *Members) save(w http.ResponseWriter, r *http.Request, id int64) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	v := func(k string) string { return strings.TrimSpace(r.PostFormValue(k)) }

	f := pages.MemberFormData{
		ID:             id,
		FullName:       v("full_name"),
		Phone:          v("phone"),
		Email:          strings.ToLower(v("email")),
		DateOfBirth:    v("date_of_birth"),
		EmergencyName:  v("emergency_name"),
		EmergencyPhone: v("emergency_phone"),
		JoinedOn:       v("joined_on"),
		Errors:         map[string]string{},
	}
	today := clock.Today()

	if f.FullName == "" {
		f.Errors["full_name"] = "Enter the member's full name."
	} else if len(f.FullName) > 100 {
		f.Errors["full_name"] = "Keep the name under 100 characters."
	}

	if !validPhone(f.Phone) {
		f.Errors["phone"] = "Enter a valid phone number."
	}
	if f.EmergencyPhone != "" && !validPhone(f.EmergencyPhone) {
		f.Errors["emergency_phone"] = "Enter a valid phone number."
	}

	if f.Email != "" {
		if a, err := mail.ParseAddress(f.Email); err != nil || a.Address != f.Email {
			f.Errors["email"] = "Enter a valid email address."
		}
	}

	var dob *time.Time
	if f.DateOfBirth != "" {
		d, err := time.Parse(dateLayout, f.DateOfBirth)
		if err != nil || !d.Before(today) || d.Year() < 1900 {
			f.Errors["date_of_birth"] = "Enter a valid date of birth."
		} else {
			dob = &d
		}
	}

	joined, err := time.Parse(dateLayout, f.JoinedOn)
	if err != nil || joined.After(today) {
		f.Errors["joined_on"] = "Enter a valid join date (not in the future)."
	}

	if len(f.Errors) > 0 {
		render(w, r, http.StatusUnprocessableEntity, pages.MemberForm(f))
		return
	}

	if id == 0 {
		_, err = h.q.CreateMember(r.Context(), db.CreateMemberParams{
			FullName:              f.FullName,
			Email:                 strPtr(f.Email),
			Phone:                 f.Phone,
			DateOfBirth:           dob,
			EmergencyContactName:  strPtr(f.EmergencyName),
			EmergencyContactPhone: strPtr(f.EmergencyPhone),
			JoinedOn:              joined,
		})
	} else {
		var n int64
		n, err = h.q.UpdateMember(r.Context(), db.UpdateMemberParams{
			ID:                    id,
			FullName:              f.FullName,
			Email:                 strPtr(f.Email),
			Phone:                 f.Phone,
			DateOfBirth:           dob,
			EmergencyContactName:  strPtr(f.EmergencyName),
			EmergencyContactPhone: strPtr(f.EmergencyPhone),
			JoinedOn:              joined,
		})
		if err == nil && n == 0 {
			http.NotFound(w, r)
			return
		}
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		f.Errors["email"] = "A member with this email already exists."
		render(w, r, http.StatusUnprocessableEntity, pages.MemberForm(f))
		return
	}
	if err != nil {
		serverError(w, "save member", err)
		return
	}
	h.respondResults(w, r, true)
}

func (h *Members) Toggle(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	m, err := h.q.GetMember(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "get member", err)
		return
	}
	if err := h.q.SetMemberActive(r.Context(), db.SetMemberActiveParams{ID: id, IsActive: !m.IsActive}); err != nil {
		serverError(w, "toggle member", err)
		return
	}
	h.respondResults(w, r, false)
}

func (h *Members) respondResults(w http.ResponseWriter, r *http.Request, closeDialog bool) {
	f := currentFilter(r)
	rows, hasNext, err := h.results(r.Context(), f)
	if err != nil {
		serverError(w, "list members", err)
		return
	}
	if closeDialog {
		w.Header().Set("HX-Retarget", "#members-results")
		w.Header().Set("HX-Reswap", "outerHTML")
		w.Header().Set("HX-Trigger", `{"close-dialog":"member-dialog"}`)
	}
	render(w, r, http.StatusOK, pages.MembersResults(f, rows, hasNext, true))
}

func validPhone(s string) bool {
	digits := 0
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9':
			digits++
		case c == '+' || c == ' ' || c == '-' || c == '(' || c == ')':
		default:
			return false
		}
	}
	return digits >= 7 && digits <= 15
}