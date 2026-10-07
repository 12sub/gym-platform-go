package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/money"
	"github.com/12sub/gym-platform/internal/permissions"
	"github.com/12sub/gym-platform/internal/views/pages"
)

type Plans struct{ q *db.Queries }

func NewPlans(q *db.Queries) *Plans { return &Plans{q: q} }

func (h *Plans) rows(ctx context.Context) ([]pages.PlanRow, error) {
	list, err := h.q.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pages.PlanRow, 0, len(list))
	for _, p := range list {
		out = append(out, pages.PlanRow{
			ID:                p.ID,
			Name:              p.Name,
			Description:       deref(p.Description),
			Price:             money.FormatNaira(p.PriceAmount),
			DurationDays:      int(p.DurationDays),
			IsActive:          p.IsActive,
			ActiveSubscribers: p.ActiveSubscribers,
		})
	}
	return out, nil
}

func (h *Plans) Index(w http.ResponseWriter, r *http.Request) {
	rows, err := h.rows(r.Context())
	if err != nil {
		serverError(w, "list plans", err)
		return
	}
	render(w, r, http.StatusOK,
		pages.PlansPage(newPage(r, "Plans", "plans"), rows, can(r, permissions.PlansManage)))
}

func (h *Plans) NewForm(w http.ResponseWriter, r *http.Request) {
	render(w, r, http.StatusOK, pages.PlanForm(pages.PlanFormData{Duration: "30"}))
}

func (h *Plans) EditForm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := h.q.GetPlan(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "get plan", err)
		return
	}
	render(w, r, http.StatusOK, pages.PlanForm(pages.PlanFormData{
		ID:          p.ID,
		Name:        p.Name,
		Description: deref(p.Description),
		Price:       money.FormatPlain(p.PriceAmount),
		Duration:    strconv.Itoa(int(p.DurationDays)),
	}))
}

func (h *Plans) Create(w http.ResponseWriter, r *http.Request) { h.save(w, r, 0) }

func (h *Plans) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.save(w, r, id)
}

func (h *Plans) save(w http.ResponseWriter, r *http.Request, id int64) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	f := pages.PlanFormData{
		ID:          id,
		Name:        strings.TrimSpace(r.PostFormValue("name")),
		Description: strings.TrimSpace(r.PostFormValue("description")),
		Price:       strings.TrimSpace(r.PostFormValue("price")),
		Duration:    strings.TrimSpace(r.PostFormValue("duration_days")),
		Errors:      map[string]string{},
	}

	if f.Name == "" {
		f.Errors["name"] = "Enter a plan name."
	} else if len(f.Name) > 80 {
		f.Errors["name"] = "Keep the name under 80 characters."
	}
	if len(f.Description) > 500 {
		f.Errors["description"] = "Keep the description under 500 characters."
	}
	price, err := money.ParseNaira(f.Price)
	if err != nil {
		f.Errors["price"] = err.Error()
	}
	days, err := strconv.Atoi(f.Duration)
	if err != nil || days < 1 || days > 3660 {
		f.Errors["duration_days"] = "Enter a number of days between 1 and 3660."
	}

	if len(f.Errors) > 0 {
		render(w, r, http.StatusUnprocessableEntity, pages.PlanForm(f))
		return
	}

	if id == 0 {
		_, err = h.q.CreatePlan(r.Context(), db.CreatePlanParams{
			Name:         f.Name,
			Description:  strPtr(f.Description),
			PriceAmount:  price,
			DurationDays: int32(days),
		})
	} else {
		var n int64
		n, err = h.q.UpdatePlan(r.Context(), db.UpdatePlanParams{
			ID:           id,
			Name:         f.Name,
			Description:  strPtr(f.Description),
			PriceAmount:  price,
			DurationDays: int32(days),
		})
		if err == nil && n == 0 {
			http.NotFound(w, r)
			return
		}
	}
	if err != nil {
		serverError(w, "save plan", err)
		return
	}
	h.respondTable(w, r, true)
}

func (h *Plans) Toggle(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	p, err := h.q.GetPlan(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "get plan", err)
		return
	}
	if err := h.q.SetPlanActive(r.Context(), db.SetPlanActiveParams{ID: id, IsActive: !p.IsActive}); err != nil {
		serverError(w, "toggle plan", err)
		return
	}
	h.respondTable(w, r, false)
}

// respondTable sends back the refreshed table. After a form save it also asks
// the browser to close the dialog.
func (h *Plans) respondTable(w http.ResponseWriter, r *http.Request, closeDialog bool) {
	rows, err := h.rows(r.Context())
	if err != nil {
		serverError(w, "list plans", err)
		return
	}
	if closeDialog {
		w.Header().Set("HX-Retarget", "#plans-table")
		w.Header().Set("HX-Reswap", "outerHTML")
		w.Header().Set("HX-Trigger", `{"close-dialog":"plan-dialog"}`)
	}
	render(w, r, http.StatusOK, pages.PlansTable(rows, true))
}