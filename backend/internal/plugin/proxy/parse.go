package proxy

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Entry is a parsed proxy line.
type Entry struct {
	Raw      string
	Scheme   string
	Host     string
	Port     int
	Username string
	Password string
}

// Display returns a redacted host:port form for logs/UI.
func (e Entry) Display() string {
	if e.Username != "" {
		return fmt.Sprintf("%s://%s:***@%s:%d", e.Scheme, e.Username, e.Host, e.Port)
	}
	return fmt.Sprintf("%s://%s:%d", e.Scheme, e.Host, e.Port)
}

// URI returns the full proxy URI (contains password; never log).
func (e Entry) URI() string {
	u := &url.URL{
		Scheme: e.Scheme,
		Host:   net.JoinHostPort(e.Host, strconv.Itoa(e.Port)),
	}
	if e.Username != "" || e.Password != "" {
		u.User = url.UserPassword(e.Username, e.Password)
	}
	return u.String()
}

// ParseLine parses one proxy line. Empty/comment lines return nil, nil.
func ParseLine(line string) (*Entry, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}
	raw := line
	if !strings.Contains(line, "://") {
		line = "http://" + line
	}
	u, err := url.Parse(line)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http", "https", "socks5", "socks5h":
	default:
		return nil, fmt.Errorf("unsupported scheme %q", scheme)
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("missing host")
	}
	portStr := u.Port()
	if portStr == "" {
		// Require explicit port for proxy lines (avoids accepting bare hostnames).
		return nil, fmt.Errorf("missing port")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port")
	}
	user, pass := "", ""
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	return &Entry{
		Raw:      raw,
		Scheme:   scheme,
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
	}, nil
}

// ParseLines parses multi-line proxy text.
func ParseLines(text string) ([]Entry, []string) {
	var out []Entry
	var errs []string
	for i, line := range strings.Split(text, "\n") {
		e, err := ParseLine(line)
		if err != nil {
			errs = append(errs, fmt.Sprintf("line %d: %v", i+1, err))
			continue
		}
		if e != nil {
			out = append(out, *e)
		}
	}
	return out, errs
}
