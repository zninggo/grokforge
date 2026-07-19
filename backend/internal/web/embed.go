package web

import (
	"embed"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
)

// Dist holds the Next.js static export (populated by `make frontend-embed`).
//
//go:embed all:dist
var Dist embed.FS

// Register mounts static UI under the root, after API routes.
func Register(e *echo.Echo) error {
	sub, err := fs.Sub(Dist, "dist")
	if err != nil {
		return err
	}

	e.GET("/*", func(c echo.Context) error {
		reqPath := c.Request().URL.Path
		if strings.HasPrefix(reqPath, "/api/") ||
			reqPath == "/healthz" || reqPath == "/readyz" ||
			reqPath == "/live" || reqPath == "/ready" {
			return echo.ErrNotFound
		}

		name := resolveStatic(sub, reqPath)
		f, err := sub.Open(name)
		if err != nil {
			// SPA-ish fallback to root index for unknown client paths.
			f, err = sub.Open("index.html")
			if err != nil {
				return echo.ErrNotFound
			}
			name = "index.html"
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return err
		}
		if stat.IsDir() {
			f.Close()
			idx := path.Join(name, "index.html")
			f, err = sub.Open(idx)
			if err != nil {
				return echo.ErrNotFound
			}
			defer f.Close()
			name = idx
			stat, err = f.Stat()
			if err != nil {
				return err
			}
		}

		ctype := mime.TypeByExtension(path.Ext(name))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		if strings.HasSuffix(name, ".html") {
			ctype = "text/html; charset=utf-8"
		}
		c.Response().Header().Set(echo.HeaderContentType, ctype)
		c.Response().WriteHeader(http.StatusOK)
		_, err = io.Copy(c.Response(), f)
		return err
	})
	return nil
}

func resolveStatic(sub fs.FS, reqPath string) string {
	clean := path.Clean("/" + reqPath)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		return "index.html"
	}
	// Prefer directory index for trailing-slash routes from Next export.
	if strings.HasSuffix(reqPath, "/") {
		candidate := clean + "/index.html"
		if okFile(sub, candidate) {
			return candidate
		}
	}
	if okFile(sub, clean) {
		return clean
	}
	// /login -> login/index.html
	candidate := clean + "/index.html"
	if okFile(sub, candidate) {
		return candidate
	}
	return clean
}

func okFile(sub fs.FS, name string) bool {
	f, err := sub.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return false
	}
	return true
}
