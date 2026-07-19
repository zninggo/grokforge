package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Account struct {
	ID           int64
	Email        string
	Upstream     string
	Status       string
	HealthStatus string
	CreatedAt    time.Time
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
		RETURNING id, email, upstream, status, health_status, created_at
	`, email, upstream, string(mb)).Scan(&a.ID, &a.Email, &a.Upstream, &a.Status, &a.HealthStatus, &a.CreatedAt)
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
	return &a, nil
}

func (r *AccountRepo) SaveEmailLease(ctx context.Context, provider, externalID, address, tokenCipher string, jobID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_leases (provider, external_id, address, token_cipher, job_id, status)
		VALUES ($1,$2,$3,$4,$5,'active')
	`, provider, externalID, address, tokenCipher, jobID)
	return err
}
