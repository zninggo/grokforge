package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminUser is the single administrator row.
type AdminUser struct {
	ID                 int64
	Username           string
	PasswordHash       string
	MustResetPassword  bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SetupState tracks installation wizard completion.
type SetupState struct {
	Completed   bool
	CompletedAt *time.Time
}

// AdminRepo persists setup + admin user.
type AdminRepo struct {
	pool *pgxpool.Pool
}

func NewAdminRepo(pool *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{pool: pool}
}

func (r *AdminRepo) GetSetupState(ctx context.Context) (*SetupState, error) {
	var st SetupState
	err := r.pool.QueryRow(ctx, `
		SELECT completed, completed_at FROM setup_state WHERE id = TRUE
	`).Scan(&st.Completed, &st.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &SetupState{Completed: false}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get setup_state: %w", err)
	}
	return &st, nil
}

func (r *AdminRepo) CompleteSetup(ctx context.Context, username, passwordHash string) (*AdminUser, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var completed bool
	if err := tx.QueryRow(ctx, `SELECT completed FROM setup_state WHERE id = TRUE FOR UPDATE`).Scan(&completed); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if _, err := tx.Exec(ctx, `INSERT INTO setup_state (id, completed) VALUES (TRUE, FALSE)`); err != nil {
				return nil, err
			}
			completed = false
		} else {
			return nil, err
		}
	}
	if completed {
		return nil, ErrSetupAlreadyDone
	}

	// Single admin: replace any existing row.
	if _, err := tx.Exec(ctx, `DELETE FROM admin_users`); err != nil {
		return nil, err
	}

	var u AdminUser
	err = tx.QueryRow(ctx, `
		INSERT INTO admin_users (username, password_hash, must_reset_password, updated_at)
		VALUES ($1, $2, FALSE, now())
		RETURNING id, username, password_hash, must_reset_password, created_at, updated_at
	`, username, passwordHash).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.MustResetPassword, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert admin: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE setup_state SET completed = TRUE, completed_at = now() WHERE id = TRUE
	`); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AdminRepo) GetByUsername(ctx context.Context, username string) (*AdminUser, error) {
	var u AdminUser
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, must_reset_password, created_at, updated_at
		FROM admin_users WHERE username = $1
	`, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.MustResetPassword, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AdminRepo) GetByID(ctx context.Context, id int64) (*AdminUser, error) {
	var u AdminUser
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, must_reset_password, created_at, updated_at
		FROM admin_users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.MustResetPassword, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *AdminRepo) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE admin_users
		SET password_hash = $2, must_reset_password = FALSE, updated_at = now()
		WHERE id = $1
	`, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Sentinel errors.
var (
	ErrNotFound         = errors.New("not found")
	ErrSetupAlreadyDone = errors.New("setup already completed")
)
