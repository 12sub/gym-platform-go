package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/12sub/gym-platform/internal/clock"
	"github.com/12sub/gym-platform/internal/db"
)

// User-facing errors: safe to show to staff as they are.
var (
	ErrMemberInactive = errors.New("This member is deactivated. Activate them first.")
	ErrPlanInactive   = errors.New("That plan is no longer available.")
	ErrFrozen         = errors.New("This membership is frozen. Unfreeze it first.")
	ErrQueued         = errors.New("A renewal is already queued. Cancel it first.")
	ErrBadState       = errors.New("That isn't possible for this subscription's current status.")
	ErrBadMethod      = errors.New("Choose how the payment was made.")
)

// ErrNotFound means the member, plan or subscription doesn't exist.
var ErrNotFound = errors.New("not found")

var userErrors = []error{ErrMemberInactive, ErrPlanInactive, ErrFrozen, ErrQueued, ErrBadState, ErrBadMethod}

func IsUserError(err error) bool {
	for _, e := range userErrors {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

type Subscriptions struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSubscriptions(pool *pgxpool.Pool, q *db.Queries) *Subscriptions {
	return &Subscriptions{pool: pool, q: q}
}

// EndDate: a plan of N days that starts on `start` ends on start+N-1 (both days count).
func EndDate(start time.Time, days int) time.Time {
	return start.AddDate(0, 0, days-1)
}

type subState struct {
	Status string
	EndsOn time.Time
}

// nextStart decides when a new subscription would begin, or why it can't.
func nextStart(current []subState, today time.Time) (start time.Time, renewal bool, err error) {
	var active *subState
	for i := range current {
		switch current[i].Status {
		case "frozen":
			return time.Time{}, false, ErrFrozen
		case "pending":
			return time.Time{}, false, ErrQueued
		case "active":
			// An "active" row that already ended is just waiting for the nightly job.
			if !current[i].EndsOn.Before(today) {
				active = &current[i]
			}
		}
	}
	if active != nil {
		return active.EndsOn.AddDate(0, 0, 1), true, nil
	}
	return today, false, nil
}

func toStates(rows []db.ListCurrentSubscriptionsRow) []subState {
	out := make([]subState, 0, len(rows))
	for _, r := range rows {
		out = append(out, subState{Status: r.Status, EndsOn: r.EndsOn})
	}
	return out
}

// NextStart previews the start date for the subscribe form.
func (s *Subscriptions) NextStart(ctx context.Context, memberID int64) (time.Time, bool, error) {
	rows, err := s.q.ListCurrentSubscriptions(ctx, memberID)
	if err != nil {
		return time.Time{}, false, err
	}
	return nextStart(toStates(rows), clock.Today())
}

func (s *Subscriptions) inTx(ctx context.Context, fn func(q *db.Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // does nothing after a successful commit
	if err := fn(s.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type SubscribeInput struct {
	MemberID   int64
	PlanID     int64
	Amount     int64 // kobo
	Method     string
	Note       string
	RecordedBy int64
}

// Subscribe creates the subscription and records its payment in one transaction.
func (s *Subscriptions) Subscribe(ctx context.Context, in SubscribeInput) error {
	switch in.Method {
	case "cash", "transfer", "pos":
	default:
		return ErrBadMethod
	}
	today := clock.Today()

	return s.inTx(ctx, func(q *db.Queries) error {
		m, err := q.LockMember(ctx, in.MemberID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !m.IsActive {
			return ErrMemberInactive
		}

		plan, err := q.GetPlan(ctx, in.PlanID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !plan.IsActive {
			return ErrPlanInactive
		}

		if err := q.ExpireOverdueForMember(ctx, db.ExpireOverdueForMemberParams{
			MemberID: in.MemberID, Today: today,
		}); err != nil {
			return err
		}

		current, err := q.ListCurrentSubscriptions(ctx, in.MemberID)
		if err != nil {
			return err
		}
		start, _, err := nextStart(toStates(current), today)
		if err != nil {
			return err
		}

		status := "active"
		if start.After(today) {
			status = "pending"
		}

		subID, err := q.CreateSubscription(ctx, db.CreateSubscriptionParams{
			MemberID: in.MemberID,
			PlanID:   in.PlanID,
			StartsOn: start,
			EndsOn:   EndDate(start, int(plan.DurationDays)),
			Status:   status,
		})
		if err != nil {
			return err
		}

		if in.Amount > 0 {
			ref, err := newReference()
			if err != nil {
				return err
			}
			note := strings.TrimSpace(in.Note)
			var notePtr *string
			if note != "" {
				notePtr = &note
			}
			return q.CreatePayment(ctx, db.CreatePaymentParams{
				MemberID:       in.MemberID,
				SubscriptionID: &subID,
				RecordedBy:     &in.RecordedBy,
				Amount:         in.Amount,
				Method:         in.Method,
				Reference:      ref,
				Note:           notePtr,
			})
		}
		return nil
	})
}

// lockBySubscription locks the owning member, then re-reads the subscription so
// the decision is made on fresh data. It returns the member ID even when the
// action is later refused, so the caller can redraw that member's page.
func lockBySubscription(ctx context.Context, q *db.Queries, subID int64) (db.Subscription, error) {
	sub, err := q.GetSubscription(ctx, subID)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Subscription{}, ErrNotFound
	}
	if err != nil {
		return db.Subscription{}, err
	}
	if _, err := q.LockMember(ctx, sub.MemberID); err != nil {
		return db.Subscription{}, err
	}
	return q.GetSubscription(ctx, subID)
}

func (s *Subscriptions) Freeze(ctx context.Context, subID int64) (memberID int64, err error) {
	today := clock.Today()
	err = s.inTx(ctx, func(q *db.Queries) error {
		sub, err := lockBySubscription(ctx, q, subID)
		if err != nil {
			return err
		}
		memberID = sub.MemberID
		if sub.Status != "active" || sub.EndsOn.Before(today) {
			return ErrBadState
		}
		return q.FreezeSubscription(ctx, db.FreezeSubscriptionParams{ID: subID, Today: today})
	})
	return memberID, err
}

func (s *Subscriptions) Unfreeze(ctx context.Context, subID int64) (memberID int64, err error) {
	today := clock.Today()
	err = s.inTx(ctx, func(q *db.Queries) error {
		sub, err := lockBySubscription(ctx, q, subID)
		if err != nil {
			return err
		}
		memberID = sub.MemberID
		if sub.Status != "frozen" {
			return ErrBadState
		}
		days := 0
		if sub.FrozenOn != nil {
			days = int(today.Sub(*sub.FrozenOn).Hours() / 24)
			if days < 0 {
				days = 0
			}
		}
		if err := q.UnfreezeSubscription(ctx, db.UnfreezeSubscriptionParams{ID: subID, Days: int32(days)}); err != nil {
			return err
		}
		// A queued renewal must move out by the same amount, or it would overlap.
		return q.ShiftPendingSubscriptions(ctx, db.ShiftPendingSubscriptionsParams{
			MemberID: sub.MemberID, Days: int32(days),
		})
	})
	return memberID, err
}

func (s *Subscriptions) Cancel(ctx context.Context, subID int64) (memberID int64, err error) {
	err = s.inTx(ctx, func(q *db.Queries) error {
		sub, err := lockBySubscription(ctx, q, subID)
		if err != nil {
			return err
		}
		memberID = sub.MemberID
		switch sub.Status {
		case "active", "frozen", "pending":
		default:
			return ErrBadState
		}
		if sub.Status != "pending" {
			// Cancelling the current one would leave the queued renewal stranded.
			current, err := q.ListCurrentSubscriptions(ctx, sub.MemberID)
			if err != nil {
				return err
			}
			for _, c := range current {
				if c.Status == "pending" {
					return ErrQueued
				}
			}
		}
		return q.CancelSubscription(ctx, subID)
	})
	return memberID, err
}

// RunDailyMaintenance expires finished subscriptions and starts queued ones
// whose start date has arrived. It is safe to run any number of times.
func (s *Subscriptions) RunDailyMaintenance(ctx context.Context) (expired, activated int64, err error) {
	today := clock.Today()
	expired, err = s.q.ExpireOverdueSubscriptions(ctx, today)
	if err != nil {
		return 0, 0, err
	}
	activated, err = s.q.ActivateDueSubscriptions(ctx, today)
	return expired, activated, err
}

func newReference() (string, error) {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "PAY-" + strings.ToUpper(hex.EncodeToString(b)), nil
}
