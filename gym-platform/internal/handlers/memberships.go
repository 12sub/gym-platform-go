package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/clock"
	"github.com/12sub/gym-platform/internal/db"
	"github.com/12sub/gym-platform/internal/money"
	"github.com/12sub/gym-platform/internal/permissions"
	"github.com/12sub/gym-platform/internal/services"
	"github.com/12sub/gym-platform/internal/views/pages"
)

const longDate = "02 Jan 2006"

var methodLabels = map[string]string{
	"cash":     "Cash",
	"transfer": "Bank transfer",
	"pos":      "POS",
	"card":     "Card",
}

type Memberships struct {
	q   *db.Queries
	svc *services.Subscriptions
}

func NewMemberships(q *db.Queries, svc *services.Subscriptions) *Memberships {
	return &Memberships{q: q, svc: svc}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// panel builds the member's profile and everything shown in the membership panel.
func (h *Memberships) panel(r *http.Request, memberID int64) (pages.MemberProfile, pages.MembershipData, error) {
	ctx := r.Context()
	today := clock.Today()

	m, err := h.q.GetMember(ctx, memberID)
	if err != nil {
		return pages.MemberProfile{}, pages.MembershipData{}, err
	}
	profile := pages.MemberProfile{
		ID:             m.ID,
		Code:           m.MemberCode,
		Name:           m.FullName,
		Email:          deref(m.Email),
		Phone:          m.Phone,
		JoinedOn:       m.JoinedOn.Format(longDate),
		EmergencyName:  deref(m.EmergencyContactName),
		EmergencyPhone: deref(m.EmergencyContactPhone),
		IsActive:       m.IsActive,
	}
	if m.DateOfBirth != nil {
		profile.DateOfBirth = m.DateOfBirth.Format(longDate)
	}

	canManage := can(r, permissions.SubscriptionsManage)
	canRecord := can(r, permissions.PaymentsRecord)
	d := pages.MembershipData{
		MemberID:     memberID,
		CanManage:    canManage,
		ShowPayments: canRecord || can(r, permissions.RevenueRead),
		Summary:      "No active membership.",
	}

	subs, err := h.q.ListMemberSubscriptions(ctx, db.ListMemberSubscriptionsParams{
		MemberID: memberID, Today: today,
	})
	if err != nil {
		return profile, d, err
	}

	var active, frozen, pending bool
	for _, s := range subs {
		row := pages.SubRow{
			ID:     s.ID,
			Plan:   s.PlanName,
			Starts: s.StartsOn.Format(longDate),
			Ends:   s.EndsOn.Format(longDate),
			Status: s.DisplayStatus,
		}
		switch s.DisplayStatus {
		case "active":
			active = true
			row.DaysLeft = int(s.EndsOn.Sub(today).Hours()/24) + 1
			d.Summary = fmt.Sprintf("%s · active until %s (%s left)", s.PlanName, row.Ends, plural(row.DaysLeft, "day", "days"))
		case "frozen":
			frozen = true
			since := ""
			if s.FrozenOn != nil {
				since = " since " + s.FrozenOn.Format(longDate)
			}
			d.Summary = fmt.Sprintf("%s · frozen%s", s.PlanName, since)
		case "pending":
			pending = true
		}
		d.Subs = append(d.Subs, row)
	}

	switch {
	case !m.IsActive:
		d.Blocked = "Activate this member before adding a subscription."
	case frozen:
		d.Blocked = "Unfreeze this membership before renewing."
	case pending:
		d.Blocked = "A renewal is already queued."
	}
	d.CanSubscribe = canManage && canRecord && d.Blocked == ""
	d.SubscribeLabel = "Add subscription"
	if active {
		d.SubscribeLabel = "Renew"
	}

	if d.ShowPayments {
		pays, err := h.q.ListMemberPayments(ctx, memberID)
		if err != nil {
			return profile, d, err
		}
		for _, p := range pays {
			date := "—"
			if p.PaidAt != nil {
				date = p.PaidAt.In(clock.Lagos).Format("02 Jan 2006, 15:04")
			}
			method := methodLabels[p.Method]
			if method == "" {
				method = p.Method
			}
			d.Payments = append(d.Payments, pages.PaymentRow{
				Date:       date,
				Plan:       p.PlanName,
				Amount:     money.FormatNaira(p.Amount),
				Method:     method,
				Status:     p.Status,
				Reference:  p.Reference,
				Note:       deref(p.Note),
				RecordedBy: p.RecordedByName,
			})
		}
	}
	return profile, d, nil
}

func (h *Memberships) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	profile, data, err := h.panel(r, id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		serverError(w, "member page", err)
		return
	}
	render(w, r, http.StatusOK, pages.MemberPage(newPage(r, profile.Name, "members"), profile, data))
}

// act runs a freeze/unfreeze/cancel and redraws the panel, with the reason on
// top if the action was refused.
func (h *Memberships) act(w http.ResponseWriter, r *http.Request, fn func(context.Context, int64) (int64, error)) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	memberID, err := fn(r.Context(), id)

	msg := ""
	switch {
	case errors.Is(err, services.ErrNotFound):
		http.NotFound(w, r)
		return
	case services.IsUserError(err):
		msg = err.Error()
	case err != nil:
		serverError(w, "subscription action", err)
		return
	}

	_, data, err := h.panel(r, memberID)
	if err != nil {
		serverError(w, "member panel", err)
		return
	}
	data.Error = msg
	status := http.StatusOK
	if msg != "" {
		status = http.StatusUnprocessableEntity
	}
	render(w, r, status, pages.MembershipPanel(data))
}

func (h *Memberships) Freeze(w http.ResponseWriter, r *http.Request)   { h.act(w, r, h.svc.Freeze) }
func (h *Memberships) Unfreeze(w http.ResponseWriter, r *http.Request) { h.act(w, r, h.svc.Unfreeze) }
func (h *Memberships) Cancel(w http.ResponseWriter, r *http.Request)   { h.act(w, r, h.svc.Cancel) }

// formBase prepares the subscribe form: the active plans and the start date.
func (h *Memberships) formBase(ctx context.Context, memberID int64) (pages.SubscriptionFormData, error) {
	f := pages.SubscriptionFormData{MemberID: memberID, Method: "cash", Errors: map[string]string{}}

	plans, err := h.q.ListPlans(ctx)
	if err != nil {
		return f, err
	}
	for _, p := range plans {
		if !p.IsActive {
			continue
		}
		f.Plans = append(f.Plans, pages.PlanOption{
			ID:    p.ID,
			Label: fmt.Sprintf("%s — %s · %d days", p.Name, money.FormatNaira(p.PriceAmount), p.DurationDays),
			Price: money.FormatPlain(p.PriceAmount),
		})
	}
	if len(f.Plans) > 0 {
		f.PlanID = strconv.FormatInt(f.Plans[0].ID, 10)
		f.Amount = f.Plans[0].Price
	}

	start, renewal, err := h.svc.NextStart(ctx, memberID)
	switch {
	case services.IsUserError(err):
		f.Errors["_"] = err.Error()
	case err != nil:
		return f, err
	default:
		f.StartsOn = start.Format(longDate)
		f.Renewal = renewal
	}
	return f, nil
}

func (h *Memberships) SubscribeForm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if _, err := h.q.GetMember(r.Context(), id); errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, "get member", err)
		return
	}
	f, err := h.formBase(r.Context(), id)
	if err != nil {
		serverError(w, "subscribe form", err)
		return
	}
	render(w, r, http.StatusOK, pages.SubscriptionForm(f))
}

func (h *Memberships) Subscribe(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	// Recording a payment is a separate permission from managing subscriptions.
	if !can(r, permissions.PaymentsRecord) {
		http.Error(w, "You don't have permission to record payments.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	f, err := h.formBase(r.Context(), id)
	if err != nil {
		serverError(w, "subscribe form", err)
		return
	}
	f.PlanID = strings.TrimSpace(r.PostFormValue("plan_id"))
	f.Amount = strings.TrimSpace(r.PostFormValue("amount"))
	f.Method = r.PostFormValue("method")
	f.Note = strings.TrimSpace(r.PostFormValue("note"))

	planID, _ := strconv.ParseInt(f.PlanID, 10, 64)
	planKnown := false
	for _, p := range f.Plans {
		if p.ID == planID {
			planKnown = true
		}
	}
	if !planKnown {
		f.Errors["plan_id"] = "Choose a plan."
	}
	amount, err := money.ParseNaira(f.Amount)
	if err != nil {
		f.Errors["amount"] = err.Error()
	}
	switch f.Method {
	case "cash", "transfer", "pos":
	default:
		f.Errors["method"] = "Choose how the payment was made."
	}
	if len(f.Note) > 200 {
		f.Errors["note"] = "Keep the note under 200 characters."
	}
	if len(f.Errors) > 0 {
		render(w, r, http.StatusUnprocessableEntity, pages.SubscriptionForm(f))
		return
	}

	u, _ := auth.UserFromContext(r.Context())
	err = h.svc.Subscribe(r.Context(), services.SubscribeInput{
		MemberID:   id,
		PlanID:     planID,
		Amount:     amount,
		Method:     f.Method,
		Note:       f.Note,
		RecordedBy: u.ID,
	})
	switch {
	case errors.Is(err, services.ErrNotFound):
		http.NotFound(w, r)
		return
	case services.IsUserError(err):
		f.Errors["_"] = err.Error()
		render(w, r, http.StatusUnprocessableEntity, pages.SubscriptionForm(f))
		return
	case err != nil:
		serverError(w, "subscribe", err)
		return
	}

	_, data, err := h.panel(r, id)
	if err != nil {
		serverError(w, "member panel", err)
		return
	}
	w.Header().Set("HX-Retarget", "#member-membership")
	w.Header().Set("HX-Reswap", "outerHTML")
	w.Header().Set("HX-Trigger", `{"close-dialog":"subscription-dialog"}`)
	render(w, r, http.StatusOK, pages.MembershipPanel(data))
}