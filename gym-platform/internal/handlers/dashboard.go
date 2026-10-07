package handlers

import (
	"net/http"

	"github.com/12sub/gym-platform/internal/views/pages"
)

type Dashboard struct{}

func NewDashboard() *Dashboard { return &Dashboard{} }

func (d *Dashboard) Show(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, pages.Dashboard(newPage(r, "Dashboard", "dashboard")))
}