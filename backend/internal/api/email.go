package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/zninggo/grokforge/internal/config"
	"github.com/zninggo/grokforge/internal/plugin/email/yyds"
)

// EmailHandler exposes YYDS config summary and connectivity test.
type EmailHandler struct {
	Cfg *config.Config
}

func (h *EmailHandler) Config(c echo.Context) error {
	return OK(c, map[string]any{
		"provider":      "yyds",
		"base_url":      yyds.DefaultBaseURL,
		"api_key_set":   h.Cfg.YYDSAPIKey != "",
		"api_key_masked": yyds.MaskKey(h.Cfg.YYDSAPIKey),
		"domain":        h.Cfg.YYDSDomain,
		"otp_defaults": map[string]any{
			"total_timeout_sec": 300,
			"long_poll_sec":     30,
			"retry_sleep_sec":   2,
		},
	})
}

func (h *EmailHandler) Test(c echo.Context) error {
	if h.Cfg.YYDSAPIKey == "" {
		return Fail(c, http.StatusBadRequest, "VALIDATION_ERROR", "GROKFORGE_YYDS_API_KEY not configured")
	}
	client := yyds.New(h.Cfg.YYDSAPIKey)
	ctx := c.Request().Context()
	domains, err := client.ListDomains(ctx)
	if err != nil {
		return Fail(c, http.StatusBadGateway, "YYDS_ERROR", err.Error())
	}
	names := make([]string, 0, len(domains))
	for _, d := range domains {
		if d.Domain != "" {
			names = append(names, d.Domain)
		}
	}
	totalDomains := len(names)
	const maxDomainPreview = 30
	if len(names) > maxDomainPreview {
		names = names[:maxDomainPreview]
	}
	// Optional: create a short-lived mailbox if domain available / configured.
	var created any
	domain := h.Cfg.YYDSDomain
	if domain == "" && len(names) > 0 {
		domain = names[0]
	}
	if domain != "" {
		acc, err := client.CreateAccount(ctx, yyds.CreateAccountRequest{Domain: domain})
		if err != nil {
			return OK(c, map[string]any{
				"ok":            true,
				"domains":       names,
				"domain_total":  totalDomains,
				"create_error":  err.Error(),
				"tested_at":     time.Now().UTC().Format(time.RFC3339),
			})
		}
		created = map[string]any{
			"id":      acc.ID,
			"address": acc.Address,
			// token intentionally omitted from response
		}
	}
	return OK(c, map[string]any{
		"ok":           true,
		"domains":      names,
		"domain_total": totalDomains,
		"account":      created,
		"tested_at":    time.Now().UTC().Format(time.RFC3339),
	})
}
