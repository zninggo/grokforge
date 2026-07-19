package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zninggo/grokforge/internal/plugin/proxy"
)

// ProxyRow is a stored proxy entry (password not returned in list DTOs).
type ProxyRow struct {
	ID             int64
	LineRaw        string
	Scheme         string
	Host           string
	Port           int
	Username       string
	PasswordCipher string
	Display        string
	Enabled        bool
	FailCount      int
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ProxyRepo struct {
	pool *pgxpool.Pool
}

func NewProxyRepo(pool *pgxpool.Pool) *ProxyRepo {
	return &ProxyRepo{pool: pool}
}

func (r *ProxyRepo) ReplaceAll(ctx context.Context, entries []proxy.Entry) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM proxy_pool`); err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		_, err := tx.Exec(ctx, `
			INSERT INTO proxy_pool (line_raw, scheme, host, port, username, password_cipher, display, enabled)
			VALUES ($1,$2,$3,$4,$5,$6,$7,TRUE)
		`, e.Raw, e.Scheme, e.Host, e.Port, e.Username, e.Password, e.Display())
		if err != nil {
			return 0, fmt.Errorf("insert proxy: %w", err)
		}
		n++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *ProxyRepo) List(ctx context.Context) ([]ProxyRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, line_raw, scheme, host, port, username, password_cipher, display,
		       enabled, fail_count, last_error, created_at, updated_at
		FROM proxy_pool
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProxyRow
	for rows.Next() {
		var p ProxyRow
		if err := rows.Scan(
			&p.ID, &p.LineRaw, &p.Scheme, &p.Host, &p.Port, &p.Username, &p.PasswordCipher, &p.Display,
			&p.Enabled, &p.FailCount, &p.LastError, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ProxyRepo) CountEnabled(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM proxy_pool WHERE enabled = TRUE`).Scan(&n)
	return n, err
}
