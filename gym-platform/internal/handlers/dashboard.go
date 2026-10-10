package handlers

import (
	"net/http"
	"strconv"

	"github.com/12sub/gym-platform/internal/clock"
	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/money"
	"github.com/12sub/gym-platform/internal/permissions"
	"github.com/12sub/gym-platform/internal/views/pages"
)

type Dashboard struct{ q *db.Queries }

func NewDashboard(q *db.Queries) *Dashboard { return &Dashboard{q: q} }

func (d *Dashboard) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var stats pages.DashboardStats

	if can(r, permissions.RevenueRead) {
		ds, de := clock.DayBounds()
		ms, me := clock.MonthBounds()

		today, err := d.q.RevenueBetween(ctx, db.RevenueBetweenParams{StartAt: ds, EndAt: de})
		if err != nil {
			serverError(w, "revenue today", err)
			return
		}
		month, err := d.q.RevenueBetween(ctx, db.RevenueBetweenParams{StartAt: ms, EndAt: me})
		if err != nil {
			serverError(w, "revenue month", err)
			return
		}
		stats.ShowRevenue = true
		stats.RevenueToday = money.FormatNaira(today)
		stats.RevenueMonth = money.FormatNaira(month)
	}

	if can(r, permissions.MembersRead) {
		active, err := d.q.CountActiveMembers(ctx)
		if err != nil {
			serverError(w, "active members", err)
			return
		}
		expiring, err := d.q.CountExpiringSoon(ctx)
		if err != nil {
			serverError(w, "expiring members", err)
			return
		}
		stats.ShowMembers = true
		stats.ActiveMembers = strconv.FormatInt(active, 10)
		stats.ExpiringSoon = strconv.FormatInt(expiring, 10)
	}

	render(w, r, http.StatusOK, pages.Dashboard(newPage(r, "Dashboard", "dashboard"), stats))
}
