package handlers

import (
	"net/http"
	"strconv"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/permissions"
	"github.com/12sub/gym-platform/internal/views/layouts"
	"github.com/12sub/gym-platform/internal/views/pages"
)

func newPage(r *http.Request, title, active string) layouts.Page {
	u, _ := auth.UserFromContext(r.Context())
	// templUI's sidebar stores "false" in this cookie when collapsed.
	collapsed := false
	if c, err := r.Cookie("sidebar_state"); err == nil && c.Value == "false" {
		collapsed = true
	}
	return layouts.Page{
		Title:            title,
		Active:           active,
		FullName:         u.FullName,
		RoleSlug:         u.RoleSlug,
		RoleLabel:        permissions.RoleLabel(u.RoleSlug),
		SidebarCollapsed: collapsed,
	}
}

func can(r *http.Request, p permissions.Permission) bool {
	u, _ := auth.UserFromContext(r.Context())
	return permissions.RoleHas(u.RoleSlug, p)
}

// isHTMX is true for partial-page requests, but not for the full-page request
// HTMX makes when restoring history after the back button.
func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true" &&
		r.Header.Get("HX-History-Restore-Request") != "true"
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ComingSoon serves a placeholder page for sections built in later steps.
func ComingSoon(title, active, blurb string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render(w, r, http.StatusOK, pages.ComingSoon(newPage(r, title, active), blurb))
	}
}