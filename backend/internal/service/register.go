package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/zninggo/grokforge/internal/config"
	"github.com/zninggo/grokforge/internal/crypto"
	"github.com/zninggo/grokforge/internal/domain"
	"github.com/zninggo/grokforge/internal/plugin/email/yyds"
	"github.com/zninggo/grokforge/internal/plugin/proxy"
	"github.com/zninggo/grokforge/internal/repository"
)

// RegisterService orchestrates mailbox + browser registration.
type RegisterService struct {
	Cfg      *config.Config
	Accounts *repository.AccountRepo
	Proxies  *repository.ProxyRepo
	Jobs     *repository.JobRepo
	Box      *crypto.Box
	Browser  domain.BrowserEngine
	// Progress is optional callback for worker event emission.
	Progress func(ctx context.Context, jobID int64, percent int, step, message string)
}

func (s *RegisterService) Run(ctx context.Context, jobID int64) error {
	report := func(p int, step, msg string) {
		if s.Progress != nil {
			s.Progress(ctx, jobID, p, step, msg)
		}
		_ = s.Jobs.UpdateProgress(ctx, jobID, p, step)
		_, _ = s.Jobs.AddEvent(ctx, jobID, "info", msg, map[string]any{"progress": p, "step": step})
	}

	report(5, "proxy", "picking proxy")
	proxyURI, proxyDisplay, err := s.pickProxy(ctx)
	if err != nil {
		return err
	}
	report(15, "proxy", "proxy selected: "+proxyDisplay)

	if s.Cfg.YYDSAPIKey == "" {
		return fmt.Errorf("YYDS_ERROR: GROKFORGE_YYDS_API_KEY not configured")
	}
	mail := yyds.New(s.Cfg.YYDSAPIKey)
	report(25, "mailbox", "creating temporary mailbox")
	acc, err := mail.CreateAccount(ctx, yyds.CreateAccountRequest{Domain: s.Cfg.YYDSDomain})
	if err != nil {
		return fmt.Errorf("YYDS_ERROR: %w", err)
	}
	tokenCipher := ""
	if s.Box != nil && acc.Token != "" {
		if c, err := s.Box.Seal(acc.Token); err == nil {
			tokenCipher = c
		}
	}
	_ = s.Accounts.SaveEmailLease(ctx, "yyds", acc.ID, acc.Address, tokenCipher, jobID)
	_ = s.Jobs.UpdateProgress(ctx, jobID, 30, "mailbox")
	// store email on job row
	_, _ = s.Jobs.AddEvent(ctx, jobID, "info", "mailbox created", map[string]any{"address": acc.Address})
	// update job email field via raw progress path — use Mark path with SQL in repo if needed
	report(35, "browser", "opening browser session")

	if s.Browser == nil {
		return fmt.Errorf("BROWSER_CRASH: no browser engine configured")
	}
	sess, err := s.Browser.Open(ctx, domain.BrowserOpenOptions{
		ProxyURI: proxyURI,
		Headless: true,
		Humanize: true,
		GeoIP:    true,
	})
	if err != nil {
		return err
	}
	defer func() { _ = sess.Close(context.Background()) }()

	report(45, "register", "starting registration flow")
	otpCfg := yyds.WaitOTPConfig{
		TotalTimeout: 300 * time.Second,
		LongPoll:     30 * time.Second,
		RetrySleep:   2 * time.Second,
	}
	result, err := sess.RegisterChatGPT(ctx, acc.Address, func(ctx context.Context) (string, error) {
		report(60, "otp", "waiting for OTP")
		code, err := mail.WaitOTP(ctx, acc.Address, otpCfg)
		if err != nil {
			return "", err
		}
		report(75, "otp", "OTP received")
		return code, nil
	})
	if err != nil {
		return err
	}
	if result.Email == "" {
		result.Email = acc.Address
	}

	report(85, "persist", "encrypting credentials")
	if s.Box == nil {
		return fmt.Errorf("MASTER_KEY required to store credentials")
	}
	blobObj := map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"cookies":       result.CookiesJSON,
		"session":       result.SessionJSON,
	}
	raw, _ := json.Marshal(blobObj)
	cipherBlob, err := s.Box.Seal(string(raw))
	if err != nil {
		return err
	}
	meta := result.Meta
	if meta == nil {
		meta = map[string]any{}
	}
	meta["job_id"] = jobID
	meta["proxy_display"] = proxyDisplay
	account, err := s.Accounts.CreateWithCredential(ctx, result.Email, result.Upstream, cipherBlob, meta)
	if err != nil {
		return err
	}
	report(100, "done", fmt.Sprintf("account saved id=%d", account.ID))
	_ = s.Jobs.MarkSucceeded(ctx, jobID, map[string]any{
		"account_id": account.ID,
		"email":      account.Email,
		"upstream":   account.Upstream,
		"dry_run":    meta["dry_run"],
	})
	return nil
}

func (s *RegisterService) pickProxy(ctx context.Context) (uri, display string, err error) {
	rows, err := s.Proxies.List(ctx)
	if err != nil {
		return "", "", err
	}
	var enabled []repository.ProxyRow
	for _, r := range rows {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	if len(enabled) == 0 {
		// allow dry-run without proxies for scaffolding; live path should configure pool
		return "", "direct", nil
	}
	r := enabled[rand.Intn(len(enabled))]
	// rebuild URI from stored parts (password_cipher currently stores plaintext until phase7 full crypto)
	e := proxy.Entry{
		Scheme:   r.Scheme,
		Host:     r.Host,
		Port:     r.Port,
		Username: r.Username,
		Password: r.PasswordCipher,
	}
	return e.URI(), r.Display, nil
}
