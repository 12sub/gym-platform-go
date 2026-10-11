package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/12sub/gym-platform/internal/auth"
	"github.com/12sub/gym-platform/internal/db"
)

var (
	ErrEmailTaken    = errors.New("Someone already uses that email address.")
	ErrSelfChange    = errors.New("You can't change your own role or deactivate yourself. Ask another owner.")
	ErrLastOwner     = errors.New("There must always be at least one active owner.")
	ErrBadRole       = errors.New("Choose a role.")
	ErrInactiveStaff = errors.New("Activate this account first.")
	ErrTokenInvalid  = errors.New("This link is invalid or has expired. Ask the owner for a new one.")
	ErrWrongPassword = errors.New("That isn't your current password.")
)

func init() {
	// These are safe to show to staff as written.
	userErrors = append(userErrors,
		ErrEmailTaken, ErrSelfChange, ErrLastOwner, ErrBadRole,
		ErrInactiveStaff, ErrTokenInvalid, ErrWrongPassword, auth.ErrWeakPassword)
}

const (
	inviteTTL = 72 * time.Hour
	resetTTL  = 24 * time.Hour

	// A password hash that can never match anything: the account exists but
	// can't be signed in to until the invite link is used.
	unusablePassword = "!"
)

type Staff struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewStaff(pool *pgxpool.Pool, q *db.Queries) *Staff {
	return &Staff{pool: pool, q: q}
}

func withTx(ctx context.Context, pool *pgxpool.Pool, q *db.Queries, fn func(q *db.Queries) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func audit(ctx context.Context, q *db.Queries, actor int64, action string, target int64, meta map[string]any) error {
	raw, _ := json.Marshal(meta)
	entity := "user"
	return q.InsertAuditLog(ctx, db.InsertAuditLogParams{
		UserID:     &actor,
		Action:     action,
		EntityType: &entity,
		EntityID:   &target,
		Metadata:   raw,
	})
}

type StaffInput struct {
	FullName string
	Email    string
	RoleSlug string
	Phone    string
	Position string
}

// Create adds the account and returns the one-time setup token (shown once).
func (s *Staff) Create(ctx context.Context, actorID int64, in StaffInput) (token string, userID int64, err error) {
	raw, hash := newToken()
	err = withTx(ctx, s.pool, s.q, func(q *db.Queries) error {
		role, err := q.GetRoleBySlug(ctx, in.RoleSlug)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBadRole
		}
		if err != nil {
			return err
		}
		id, err := q.CreateUser(ctx, db.CreateUserParams{
			FullName:     in.FullName,
			Email:        in.Email,
			PasswordHash: unusablePassword,
			RoleID:       role.ID,
		})
		if isUniqueViolation(err) {
			return ErrEmailTaken
		}
		if err != nil {
			return err
		}
		userID = id

		if err := q.UpsertStaffProfile(ctx, db.UpsertStaffProfileParams{
			UserID: id, Phone: ptr(in.Phone), Position: ptr(in.Position),
		}); err != nil {
			return err
		}
		if err := q.CreatePasswordToken(ctx, db.CreatePasswordTokenParams{
			UserID:    id,
			TokenHash: hash,
			Purpose:   "invite",
			ExpiresAt: time.Now().Add(inviteTTL),
			CreatedBy: &actorID,
		}); err != nil {
			return err
		}
		return audit(ctx, q, actorID, "staff.created", id, map[string]any{"email": in.Email, "role": in.RoleSlug})
	})
	if err != nil {
		return "", 0, err
	}
	return raw, userID, nil
}

// Update changes name, role and contact details. Email is the account's
// identity and isn't changed here.
func (s *Staff) Update(ctx context.Context, actorID, targetID int64, in StaffInput) error {
	return withTx(ctx, s.pool, s.q, func(q *db.Queries) error {
		if err := q.LockStaffChanges(ctx); err != nil {
			return err
		}
		cur, err := q.GetStaff(ctx, targetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		role, err := q.GetRoleBySlug(ctx, in.RoleSlug)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBadRole
		}
		if err != nil {
			return err
		}

		if cur.RoleSlug != in.RoleSlug {
			if targetID == actorID {
				return ErrSelfChange
			}
			if cur.RoleSlug == "owner" && cur.IsActive {
				n, err := q.CountActiveOwnersExcept(ctx, targetID)
				if err != nil {
					return err
				}
				if n == 0 {
					return ErrLastOwner
				}
			}
		}

		rows, err := q.UpdateStaffUser(ctx, db.UpdateStaffUserParams{
			ID: targetID, FullName: in.FullName, RoleID: role.ID,
		})
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrNotFound
		}
		if err := q.UpsertStaffProfile(ctx, db.UpsertStaffProfileParams{
			UserID: targetID, Phone: ptr(in.Phone), Position: ptr(in.Position),
		}); err != nil {
			return err
		}
		return audit(ctx, q, actorID, "staff.updated", targetID, map[string]any{
			"role_from": cur.RoleSlug, "role_to": in.RoleSlug,
		})
	})
}

// SetActive deactivates or reactivates an account. A deactivated person is
// signed out on their next click, because every request re-checks the account.
func (s *Staff) SetActive(ctx context.Context, actorID, targetID int64, active bool) error {
	return withTx(ctx, s.pool, s.q, func(q *db.Queries) error {
		if err := q.LockStaffChanges(ctx); err != nil {
			return err
		}
		cur, err := q.GetStaff(ctx, targetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !active {
			if targetID == actorID {
				return ErrSelfChange
			}
			if cur.RoleSlug == "owner" && cur.IsActive {
				n, err := q.CountActiveOwnersExcept(ctx, targetID)
				if err != nil {
					return err
				}
				if n == 0 {
					return ErrLastOwner
				}
			}
		}
		if err := q.SetUserActive(ctx, db.SetUserActiveParams{ID: targetID, IsActive: active}); err != nil {
			return err
		}
		action := "staff.deactivated"
		if active {
			action = "staff.activated"
		}
		return audit(ctx, q, actorID, action, targetID, map[string]any{"email": cur.Email})
	})
}

// IssueLink makes a fresh setup link: an invite for someone who hasn't set a
// password yet, otherwise a password reset. Any earlier link stops working.
func (s *Staff) IssueLink(ctx context.Context, actorID, targetID int64) (token, purpose string, ttl time.Duration, err error) {
	raw, hash := newToken()
	err = withTx(ctx, s.pool, s.q, func(q *db.Queries) error {
		cur, err := q.GetStaff(ctx, targetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !cur.IsActive {
			return ErrInactiveStaff
		}
		purpose, ttl = "reset", resetTTL
		if cur.PendingSetup {
			purpose, ttl = "invite", inviteTTL
		}
		if err := q.InvalidateUserTokens(ctx, targetID); err != nil {
			return err
		}
		if err := q.CreatePasswordToken(ctx, db.CreatePasswordTokenParams{
			UserID: targetID, TokenHash: hash, Purpose: purpose,
			ExpiresAt: time.Now().Add(ttl), CreatedBy: &actorID,
		}); err != nil {
			return err
		}
		return audit(ctx, q, actorID, "staff.link_issued", targetID, map[string]any{"purpose": purpose})
	})
	if err != nil {
		return "", "", 0, err
	}
	return raw, purpose, ttl, nil
}

type TokenInfo struct {
	TokenID  int64
	UserID   int64
	Purpose  string
	FullName string
	Email    string
}

func (s *Staff) LookupToken(ctx context.Context, raw string) (TokenInfo, error) {
	row, err := s.q.GetUsablePasswordToken(ctx, hashToken(raw))
	if errors.Is(err, pgx.ErrNoRows) {
		return TokenInfo{}, ErrTokenInvalid
	}
	if err != nil {
		return TokenInfo{}, err
	}
	if !row.IsActive {
		return TokenInfo{}, ErrTokenInvalid
	}
	return TokenInfo{
		TokenID: row.ID, UserID: row.UserID, Purpose: row.Purpose,
		FullName: row.FullName, Email: row.Email,
	}, nil
}

// CompleteSetup uses a link once: it saves the new password, signs out any
// existing sessions, and retires every other link for that account.
func (s *Staff) CompleteSetup(ctx context.Context, raw, newPassword string) (TokenInfo, error) {
	info, err := s.LookupToken(ctx, raw)
	if err != nil {
		return info, err
	}
	if err := auth.ValidateNewPassword(newPassword, info.Email); err != nil {
		return info, err
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return info, err
	}
	err = withTx(ctx, s.pool, s.q, func(q *db.Queries) error {
		used, err := q.MarkPasswordTokenUsed(ctx, info.TokenID)
		if err != nil {
			return err
		}
		if used == 0 {
			return ErrTokenInvalid // someone else used it a moment ago
		}
		if _, err := q.SetPassword(ctx, db.SetPasswordParams{ID: info.UserID, PasswordHash: hash}); err != nil {
			return err
		}
		if err := q.InvalidateUserTokens(ctx, info.UserID); err != nil {
			return err
		}
		return audit(ctx, q, info.UserID, "password.set_via_link", info.UserID, map[string]any{"purpose": info.Purpose})
	})
	return info, err
}

// ChangePassword lets signed-in users change their own password. It returns the
// new token version so the caller can keep the current session alive.
func (s *Staff) ChangePassword(ctx context.Context, userID int64, email, current, next string) (int32, error) {
	stored, err := s.q.GetPasswordHash(ctx, userID)
	if err != nil {
		return 0, err
	}
	if ok, cerr := auth.CheckPassword(current, stored); cerr != nil || !ok {
		return 0, ErrWrongPassword
	}
	if err := auth.ValidateNewPassword(next, email); err != nil {
		return 0, err
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return 0, err
	}

	var version int32
	err = withTx(ctx, s.pool, s.q, func(q *db.Queries) error {
		v, err := q.SetPassword(ctx, db.SetPasswordParams{ID: userID, PasswordHash: hash})
		if err != nil {
			return err
		}
		version = v
		return audit(ctx, q, userID, "password.changed", userID, map[string]any{})
	})
	return version, err
}