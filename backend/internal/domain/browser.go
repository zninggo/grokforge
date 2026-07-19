package domain

import "context"

// BrowserSession is an open automation session.
type BrowserSession interface {
	// RegisterChatGPT runs the human-like signup flow for the given email.
	// otpFetcher is called when the page needs a verification code.
	RegisterChatGPT(ctx context.Context, email string, otpFetcher func(context.Context) (string, error)) (*RegisterResult, error)
	Close(ctx context.Context) error
}

// BrowserEngine opens browser sessions (Cloak free binary, etc.).
type BrowserEngine interface {
	Name() string
	Available() bool
	Open(ctx context.Context, opts BrowserOpenOptions) (BrowserSession, error)
}

// BrowserOpenOptions configures a single registration browser.
type BrowserOpenOptions struct {
	ProxyURI  string
	Headless  bool
	Humanize  bool
	GeoIP     bool
	UserAgent string
}

// RegisterResult is the durable outcome of a successful registration.
type RegisterResult struct {
	Email    string
	Upstream string
	// Secrets are never logged; stored encrypted by account service.
	AccessToken  string
	RefreshToken string
	CookiesJSON  string
	SessionJSON  string
	Meta         map[string]any
}
