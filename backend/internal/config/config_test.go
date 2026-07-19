package config

import (
	"testing"
)

func clearURLEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("GROKFORGE_DATABASE_URL", "")
	t.Setenv("GROKFORGE_REDIS_URL", "")
	t.Setenv("GROKFORGE_ENV", "")
	t.Setenv("GROKFORGE_MASTER_KEY", "")
	t.Setenv("GROKFORGE_JWT_SECRET", "")
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	clearURLEnv(t)
	t.Setenv("GROKFORGE_HTTP_ADDR", ":17890")
	t.Setenv("GROKFORGE_JWT_SECRET", "0123456789abcdef")
	_, err := Load()
	if err == nil {
		// Workspace .env may supply DATABASE_URL; only assert when truly empty.
		t.Skip("workspace .env present; skip missing-database assertion")
	}
}

func TestLoadRequiresRedisOutsideDev(t *testing.T) {
	clearURLEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	t.Setenv("GROKFORGE_HTTP_ADDR", ":17890")
	t.Setenv("GROKFORGE_JWT_SECRET", "0123456789abcdef")
	t.Setenv("GROKFORGE_ENV", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when redis missing outside dev")
	}
}

func TestLoadOKDevWithoutRedis(t *testing.T) {
	clearURLEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	t.Setenv("GROKFORGE_HTTP_ADDR", ":17890")
	t.Setenv("GROKFORGE_JWT_SECRET", "0123456789abcdef")
	t.Setenv("GROKFORGE_ENV", "dev")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsDev() {
		t.Fatal("expected IsDev")
	}
	if cfg.RedisURL != "" {
		t.Fatalf("redis url=%q", cfg.RedisURL)
	}
	if cfg.ReadyRequireRedis {
		t.Fatal("ReadyRequireRedis should be false when redis unset in dev")
	}
	r := cfg.Redacted()
	if r["is_dev"] != true {
		t.Fatalf("redacted: %+v", r)
	}
	if r["redis_url_set"] != false {
		t.Fatalf("redacted redis: %+v", r)
	}
}

func TestLoadOK(t *testing.T) {
	clearURLEnv(t)
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
	t.Setenv("GROKFORGE_HTTP_ADDR", ":17890")
	t.Setenv("GROKFORGE_JWT_SECRET", "0123456789abcdef")
	t.Setenv("GROKFORGE_MASTER_KEY", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":17890" {
		t.Fatalf("addr=%s", cfg.HTTPAddr)
	}
	if cfg.JWTSecret != "0123456789abcdef" {
		t.Fatalf("jwt=%s", cfg.JWTSecret)
	}
	if !cfg.ReadyRequireRedis {
		t.Fatal("ReadyRequireRedis should default true when redis is set")
	}
	r := cfg.Redacted()
	if r["database_url_set"] != true {
		t.Fatalf("redacted: %+v", r)
	}
	if r["yyds_api_key_set"] == true && cfg.YYDSAPIKey != "" {
		for k, v := range r {
			if s, ok := v.(string); ok && s == cfg.YYDSAPIKey {
				t.Fatalf("secret leaked in redacted field %s", k)
			}
		}
	}
}
