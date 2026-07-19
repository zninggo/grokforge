package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditLog struct {
	ID         int64
	Actor      string
	Action     string
	TargetType string
	TargetID   string
	Detail     []byte
	CreatedAt  time.Time
}

type AuditRepo struct {
	pool *pgxpool.Pool
}

func NewAuditRepo(pool *pgxpool.Pool) *AuditRepo {
	return &AuditRepo{pool: pool}
}

func (r *AuditRepo) Add(ctx context.Context, actor, action, targetType, targetID string, detail map[string]any) error {
	if actor == "" {
		actor = "admin"
	}
	b, _ := json.Marshal(detail)
	if detail == nil {
		b = []byte("{}")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_logs (actor, action, target_type, target_id, detail)
		VALUES ($1,$2,$3,$4,$5::jsonb)
	`, actor, action, targetType, targetID, string(b))
	return err
}

func (r *AuditRepo) List(ctx context.Context, limit, offset int) ([]AuditLog, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor, action, target_type, target_id, detail, created_at
		FROM audit_logs
		ORDER BY id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []AuditLog
	for rows.Next() {
		var a AuditLog
		if err := rows.Scan(&a.ID, &a.Actor, &a.Action, &a.TargetType, &a.TargetID, &a.Detail, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}
