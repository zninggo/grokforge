package yyds

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://maliapi.215.im"

// Client talks to YYDS Mail API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

type Domain struct {
	Domain string `json:"domain"`
}

type Account struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

type Message struct {
	ID               string `json:"id"`
	From             string `json:"from"`
	Subject          string `json:"subject"`
	VerificationCode string `json:"verificationCode"`
}

type apiEnvelope struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data"`
	Error     string          `json:"error"`
	ErrorCode string          `json:"errorCode"`
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) (int, error) {
	if c.APIKey == "" {
		return 0, fmt.Errorf("yyds api key empty")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	u := base + path
	if query != nil {
		u += "?" + query.Encode()
	}
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-API-Key", c.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode == http.StatusNoContent {
		return resp.StatusCode, nil
	}
	if len(raw) == 0 {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp.StatusCode, nil
		}
		return resp.StatusCode, fmt.Errorf("yyds http %d empty body", resp.StatusCode)
	}

	// Try envelope first.
	var env apiEnvelope
	if err := json.Unmarshal(raw, &env); err == nil && (env.Data != nil || env.Error != "" || env.ErrorCode != "" || env.Success) {
		if !env.Success && env.Error != "" {
			return resp.StatusCode, fmt.Errorf("yyds: %s (%s)", env.Error, env.ErrorCode)
		}
		if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return resp.StatusCode, fmt.Errorf("decode data: %w", err)
			}
		}
		if resp.StatusCode >= 400 {
			return resp.StatusCode, fmt.Errorf("yyds http %d", resp.StatusCode)
		}
		return resp.StatusCode, nil
	}

	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode: %w; body=%s", err, truncate(string(raw), 200))
		}
	}
	if resp.StatusCode >= 400 {
		return resp.StatusCode, fmt.Errorf("yyds http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	return resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (c *Client) ListDomains(ctx context.Context) ([]Domain, error) {
	var domains []Domain
	// data may be array or object with items
	var raw json.RawMessage
	code, err := c.do(ctx, http.MethodGet, "/v1/domains", nil, nil, &raw)
	if err != nil {
		return nil, err
	}
	if code == http.StatusNoContent {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &domains); err == nil {
		return domains, nil
	}
	var wrap struct {
		Items   []Domain `json:"items"`
		Domains []Domain `json:"domains"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Items) > 0 {
		return wrap.Items, nil
	}
	return wrap.Domains, nil
}

type CreateAccountRequest struct {
	LocalPart string `json:"localPart,omitempty"`
	Domain    string `json:"domain,omitempty"`
	Subdomain string `json:"subdomain,omitempty"`
}

func (c *Client) CreateAccount(ctx context.Context, req CreateAccountRequest) (*Account, error) {
	var acc Account
	var raw json.RawMessage
	if _, err := c.do(ctx, http.MethodPost, "/v1/accounts", nil, req, &raw); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &acc); err != nil {
		return nil, err
	}
	if acc.Address == "" {
		return nil, fmt.Errorf("yyds create account: empty address")
	}
	return &acc, nil
}

// NextMessage long-polls for the next message. 204 means none yet.
func (c *Client) NextMessage(ctx context.Context, address string, waitSec int) (*Message, bool, error) {
	if waitSec < 0 {
		waitSec = 0
	}
	if waitSec > 30 {
		waitSec = 30
	}
	q := url.Values{}
	q.Set("address", address)
	q.Set("wait", fmt.Sprintf("%d", waitSec))
	var raw json.RawMessage
	code, err := c.do(ctx, http.MethodGet, "/v1/messages/next", q, nil, &raw)
	if err != nil {
		return nil, false, err
	}
	if code == http.StatusNoContent || len(raw) == 0 || string(raw) == "null" {
		return nil, false, nil
	}
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, false, err
	}
	if msg.VerificationCode == "" && msg.ID == "" {
		return nil, false, nil
	}
	return &msg, true, nil
}

// WaitOTPConfig controls WaitOTP.
type WaitOTPConfig struct {
	TotalTimeout time.Duration
	LongPoll     time.Duration
	RetrySleep   time.Duration
	// Now and Sleep allow tests to inject clocks.
	Now   func() time.Time
	Sleep func(context.Context, time.Duration) error
	// Next is optional override for NextMessage (tests).
	Next func(ctx context.Context, address string, waitSec int) (*Message, bool, error)
}

// WaitOTP polls until verificationCode is available or total timeout.
func (c *Client) WaitOTP(ctx context.Context, address string, cfg WaitOTPConfig) (string, error) {
	if cfg.TotalTimeout <= 0 {
		cfg.TotalTimeout = 300 * time.Second
	}
	if cfg.LongPoll <= 0 {
		cfg.LongPoll = 30 * time.Second
	}
	if cfg.RetrySleep < 0 {
		cfg.RetrySleep = 0
	}
	if cfg.RetrySleep == 0 {
		cfg.RetrySleep = 2 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Sleep == nil {
		cfg.Sleep = func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	next := cfg.Next
	if next == nil {
		next = c.NextMessage
	}

	deadline := cfg.Now().Add(cfg.TotalTimeout)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		remain := deadline.Sub(cfg.Now())
		if remain <= 0 {
			return "", ErrOTPTimeout
		}
		wait := cfg.LongPoll
		if remain < wait {
			wait = remain
		}
		waitSec := int(wait.Seconds())
		if waitSec < 1 {
			waitSec = 1
		}
		if waitSec > 30 {
			waitSec = 30
		}
		msg, ok, err := next(ctx, address, waitSec)
		if err != nil {
			return "", err
		}
		if ok && msg != nil && strings.TrimSpace(msg.VerificationCode) != "" {
			return strings.TrimSpace(msg.VerificationCode), nil
		}
		if cfg.Now().After(deadline) || cfg.Now().Equal(deadline) {
			return "", ErrOTPTimeout
		}
		sleep := cfg.RetrySleep
		if left := deadline.Sub(cfg.Now()); left < sleep {
			sleep = left
		}
		if sleep > 0 {
			if err := cfg.Sleep(ctx, sleep); err != nil {
				return "", err
			}
		}
	}
}

// ErrOTPTimeout is returned when OTP is not received in time.
var ErrOTPTimeout = fmt.Errorf("OTP_TIMEOUT")

// MaskKey redacts an API key for display.
func MaskKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
