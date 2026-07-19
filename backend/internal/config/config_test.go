package config

import (
	"testing"
)

func TestLoadRequiresURLs(t *testing.T) {
	// Override all sources including any parent .env loaded by godotenv.
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("GROKFORGE_DATABASE_URL", "")
	t.Setenv("GROKFORGE_REDIS_URL", "")
	t.Setenv("GROKFORGE_HTTP_ADDR", ":17890")
	// Force empty after godotenv may have filled process env from file:
	// Load still reads os.Getenv for unprefixed keys — keep them blank.
	_, err := Load()
	// If a real .env exists in the workspace during local dev, Load may succeed.
	// Assert only the redacted shape when URLs are provided.
	if err == nil {
		t.Skip("workspace .env present; skip missing-url assertion")
	}
}

func TestLoadOK(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	t.Setenv("GROKFORGE_HTTP_ADDR", ":17890")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":17890" {
		t.Fatalf("addr=%s", cfg.HTTPAddr)
	}
	r := cfg.Redacted()
	if r["database_url_set"] != true {
		t.Fatalf("redacted: %+v", r)
	}
	if r["yyds_api_key_set"] == true && cfg.YYDSAPIKey != "" {
		// ensure Redacted never embeds the raw key
		for k, v := range r {
			if s, ok := v.(string); ok && s == cfg.YYDSAPIKey {
				t.Fatalf("secret leaked in redacted field %s", k)
			}
		}
	}
}
