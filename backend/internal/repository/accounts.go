package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Account struct {
	ID           int64
	Email        string
	Upstream     string
	Status       string
	HealthStatus string
	HealthDetail string
	PlanSnapshot []byte
	Meta         []byte
	LastProbedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	HasCredential bool
}

type AccountRepo struct {
	pool *pgxpool.Pool
}

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

func (r *AccountRepo) CreateWithCredential(ctx context.Context, email, upstream, cipherBlob string, meta map[string]any) (*Account, error) {
	if upstream == "" {
		upstream = "chatgpt"
	}
	mb, _ := json.Marshal(meta)
	if meta == nil {
		mb = []byte("{}")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var a Account
	err = tx.QueryRow(ctx, `
		INSERT INTO accounts (email, upstream, status, health_status, meta)
		VALUES ($1, $2, 'active', 'unknown', $3::jsonb)
		ON CONFLICT (upstream, email) DO UPDATE
		  SET meta = EXCLUDED.meta, updated_at = now(), status = 'active'
		RETURNING id, email, upstream, status, health_status, health_detail, plan_snapshot, meta,
		          last_probed_at, created_at, updated_at
	`, email, upstream, string(mb)).Scan(
		&a.ID, &a.Email, &a.Upstream, &a.Status, &a.HealthStatus, &a.HealthDetail, &a.PlanSnapshot, &a.Meta,
		&a.LastProbedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert account: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO credentials (account_id, cipher_blob, key_version)
		VALUES ($1, $2, 1)
		ON CONFLICT (account_id) DO UPDATE
		  SET cipher_blob = EXCLUDED.cipher_blob, updated_at = now()
	`, a.ID, cipherBlob)
	if err != nil {
		return nil, fmt.Errorf("insert credential: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	a.HasCredential = true
	return &a, nil
}

func (r *AccountRepo) SaveEmailLease(ctx context.Context, provider, externalID, address, tokenCipher string, jobID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_leases (provider, external_id, address, token_cipher, job_id, status)
		VALUES ($1,$2,$3,$4,$5,'active')
	`, provider, externalID, address, tokenCipher, jobID)
	return err
}

func (r *AccountRepo) List(ctx context.Context, health string, limit, offset int) ([]Account, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM accounts WHERE ($1 = '' OR health_status = $1)
	`, health).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.email, a.upstream, a.status, a.health_status, a.health_detail,
		       a.plan_snapshot, a.meta, a.last_probed_at, a.created_at, a.updated_at,
		       EXISTS(SELECT 1 FROM credentials c WHERE c.account_id = a.id) AS has_cred
		FROM accounts a
		WHERE ($1 = '' OR a.health_status = $1)
		ORDER BY a.id DESC
		LIMIT $2 OFFSET $3
	`, health, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(
			&a.ID, &a.Email, &a.Upstream, &a.Status, &a.HealthStatus, &a.HealthDetail,
			&a.PlanSnapshot, &a.Meta, &a.LastProbedAt, &a.CreatedAt, &a.UpdatedAt, &a.HasCredential,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

func (r *AccountRepo) Get(ctx context.Context, id int64) (*Account, error) {
	var a Account
	err := r.pool.QueryRow(ctx, `
		SELECT a.id, a.email, a.upstream, a.status, a.health_status, a.health_detail,
		       a.plan_snapshot, a.meta, a.last_probed_at, a.created_at, a.updated_at,
		       EXISTS(SELECT 1 FROM credentials c WHERE c.account_id = a.id) AS has_cred
		FROM accounts a WHERE a.id = $1
	`, id).Scan(
		&a.ID, &a.Email, &a.Upstream, &a.Status, &a.HealthStatus, &a.HealthDetail,
		&a.PlanSnapshot, &a.Meta, &a.LastProbedAt, &a.CreatedAt, &a.UpdatedAt, &a.HasCredential,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AccountRepo) GetCredentialCipher(ctx context.Context, accountID int64) (string, int, error) {
	var blob string
	var ver int
	err := r.pool.QueryRow(ctx, `
		SELECT cipher_blob, key_version FROM credentials WHERE account_id = $1
	`, accountID).Scan(&blob, &ver)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, ErrNotFound
	}
	return blob, ver, err
}

func (r *AccountRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM accounts WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AccountRepo) DeleteBatch(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM accounts WHERE id = ANY($1)`, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *AccountRepo) UpdateHealth(ctx context.Context, id int64, status, detail string, plan map[string]any) error {
	b, _ := json.Marshal(plan)
	if plan == nil {
		b = []byte("{}")
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE accounts
		SET health_status = $2, health_detail = $3, plan_snapshot = $4::jsonb,
		    last_probed_at = now(), updated_at = now()
		WHERE id = $1
	`, id, status, detail, string(b))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *AccountRepo) ListIDs(ctx context.Context, onlyUnknown bool) ([]int64, error) {
	q := `SELECT id FROM accounts`
	if onlyUnknown {
		q += ` WHERE health_status = 'unknown'`
	}
	q += ` ORDER BY id ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *AccountRepo) Count(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM accounts`).Scan(&n)
	return n, err
}
