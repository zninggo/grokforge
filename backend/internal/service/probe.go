package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zninggo/grokforge/internal/crypto"
	"github.com/zninggo/grokforge/internal/repository"
)

// ProbeService checks account session health without chat completions.
type ProbeService struct {
	Accounts *repository.AccountRepo
	Box      *crypto.Box
}

// ErrNoMasterKey when decrypt is required but master key missing.
var ErrNoMasterKey = fmt.Errorf("master key not configured")

type ProbeResult struct {
	AccountID    int64          `json:"account_id"`
	Email        string         `json:"email"`
	HealthStatus string         `json:"health_status"`
	Detail       string         `json:"detail"`
	PlanSnapshot map[string]any `json:"plan_snapshot"`
}

// ProbeAccount inspects stored credentials only (no chat API).
// Rules:
//   - decrypt fails → restricted / network_error style detail
//   - missing access_token → expired
//   - dry_run tokens → alive with dry_run flag (scaffold)
//   - otherwise → alive with session_present (live session HTTP check is Phase 8 extension)
func (s *ProbeService) ProbeAccount(ctx context.Context, accountID int64) (*ProbeResult, error) {
	a, err := s.Accounts.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if s.Box == nil {
		return nil, ErrNoMasterKey
	}
	cipher, _, err := s.Accounts.GetCredentialCipher(ctx, accountID)
	if err != nil {
		_ = s.Accounts.UpdateHealth(ctx, accountID, "restricted", "no credential", nil)
		return &ProbeResult{AccountID: accountID, Email: a.Email, HealthStatus: "restricted", Detail: "no credential"}, nil
	}
	pt, err := s.Box.Open(cipher)
	if err != nil {
		_ = s.Accounts.UpdateHealth(ctx, accountID, "restricted", "decrypt failed", nil)
		return nil, fmt.Errorf("decrypt failed: %w", err)
	}
	var cred map[string]any
	_ = json.Unmarshal([]byte(pt), &cred)
	access, _ := cred["access_token"].(string)
	access = strings.TrimSpace(access)

	status := "alive"
	detail := "session credential present"
	plan := map[string]any{"source": "local_credential"}

	if access == "" {
		status = "expired"
		detail = "access_token empty"
	} else if strings.HasPrefix(access, "dry_run_") {
		status = "alive"
		detail = "dry_run credential"
		plan["dry_run"] = true
	}

	// Explicit non-goals: never call chat completions / messages / responses endpoints.
	if err := s.Accounts.UpdateHealth(ctx, accountID, status, detail, plan); err != nil {
		return nil, err
	}
	return &ProbeResult{
		AccountID:    accountID,
		Email:        a.Email,
		HealthStatus: status,
		Detail:       detail,
		PlanSnapshot: plan,
	}, nil
}
