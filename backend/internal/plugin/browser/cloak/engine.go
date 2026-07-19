package cloak

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/zninggo/grokforge/internal/domain"
)

// Engine drives CloakBrowser free binary (license empty).
type Engine struct {
	// Binary is the path to cloak executable. Empty → auto-detect.
	Binary string
	// DryRun forces a no-browser simulation path (tests / no binary).
	DryRun bool
}

func (e *Engine) Name() string { return "chatgpt.cloak" }

func (e *Engine) Available() bool {
	if e.DryRun {
		return true
	}
	return e.resolveBinary() != ""
}

func (e *Engine) resolveBinary() string {
	if e.Binary != "" {
		if st, err := os.Stat(e.Binary); err == nil && !st.IsDir() {
			return e.Binary
		}
	}
	if p := os.Getenv("GROKFORGE_CLOAK_BIN"); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	candidates := []string{
		"cloak",
		"cloakbrowser",
		"/usr/local/bin/cloak",
		"/opt/cloak/cloak",
		filepath.Join("bin", "cloak"),
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func (e *Engine) Open(ctx context.Context, opts domain.BrowserOpenOptions) (domain.BrowserSession, error) {
	if e.DryRun || strings.EqualFold(os.Getenv("GROKFORGE_CLOAK_DRY_RUN"), "1") {
		return &drySession{opts: opts}, nil
	}
	bin := e.resolveBinary()
	if bin == "" {
		return nil, fmt.Errorf("BROWSER_CRASH: cloak binary not found (set GROKFORGE_CLOAK_BIN or install free Cloak binary)")
	}
	// Phase 6 foundation: binary presence check + session shell.
	// Full CDP/Playwright binding lands when binary is present in deploy image.
	return &liveSession{
		bin:  bin,
		opts: opts,
	}, nil
}

type drySession struct {
	opts domain.BrowserOpenOptions
}

func (s *drySession) RegisterChatGPT(ctx context.Context, email string, otpFetcher func(context.Context) (string, error)) (*domain.RegisterResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}
	// Dry-run does not call real OTP (no mail sent); pipeline tests mailbox create separately.
	_ = otpFetcher
	return &domain.RegisterResult{
		Email:        email,
		Upstream:     "chatgpt",
		AccessToken:  "dry_run_access",
		RefreshToken: "dry_run_refresh",
		CookiesJSON:  "[]",
		SessionJSON:  `{"dry_run":true}`,
		Meta: map[string]any{
			"driver":  "chatgpt.cloak",
			"dry_run": true,
			"proxy":   redactProxy(s.opts.ProxyURI),
		},
	}, nil
}

func (s *drySession) Close(ctx context.Context) error { return nil }

type liveSession struct {
	bin  string
	opts domain.BrowserOpenOptions
	cmd  *exec.Cmd
}

func (s *liveSession) RegisterChatGPT(ctx context.Context, email string, otpFetcher func(context.Context) (string, error)) (*domain.RegisterResult, error) {
	// Live automation requires Cloak runtime binding (CDP/extension protocol).
	// Until the binary-specific protocol is wired in the deploy image, fail clearly.
	_ = email
	_ = otpFetcher
	return nil, fmt.Errorf("REGISTER_FAILED: cloak live automation not bound yet for binary %s (use dry_run or complete binding in deploy image)", s.bin)
}

func (s *liveSession) Close(ctx context.Context) error {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	return nil
}

func redactProxy(uri string) string {
	if uri == "" {
		return ""
	}
	if i := strings.Index(uri, "://"); i >= 0 {
		rest := uri[i+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			return uri[:i+3] + "***@" + rest[at+1:]
		}
	}
	return uri
}
