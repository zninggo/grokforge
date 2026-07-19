package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds process configuration loaded from env / optional .env file.
type Config struct {
	HTTPAddr    string
	DatabaseURL string
	RedisURL    string
	MasterKey   string
	JWTSecret   string
	YYDSAPIKey  string
	YYDSDomain  string
	LogLevel    string
	// ReadyRequireRedis when true makes /readyz fail if Redis is down.
	ReadyRequireRedis bool
}

// Load reads configuration. It optionally loads a .env file from cwd or parent
// directories (never required). Environment variables always win.
func Load() (*Config, error) {
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("../../.env")

	v := viper.New()
	v.SetEnvPrefix("GROKFORGE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Also accept unprefixed standard URLs used by compose / PaaS.
	_ = v.BindEnv("database_url", "GROKFORGE_DATABASE_URL", "DATABASE_URL")
	_ = v.BindEnv("redis_url", "GROKFORGE_REDIS_URL", "REDIS_URL")
	_ = v.BindEnv("http_addr", "GROKFORGE_HTTP_ADDR")
	_ = v.BindEnv("master_key", "GROKFORGE_MASTER_KEY")
	_ = v.BindEnv("jwt_secret", "GROKFORGE_JWT_SECRET")
	_ = v.BindEnv("yyds_api_key", "GROKFORGE_YYDS_API_KEY")
	_ = v.BindEnv("yyds_domain", "GROKFORGE_YYDS_DOMAIN")
	_ = v.BindEnv("log_level", "GROKFORGE_LOG_LEVEL")

	v.SetDefault("http_addr", ":17890")
	v.SetDefault("log_level", "info")
	v.SetDefault("ready_require_redis", true)

	cfg := &Config{
		HTTPAddr:          v.GetString("http_addr"),
		DatabaseURL:       firstNonEmpty(v.GetString("database_url"), os.Getenv("DATABASE_URL")),
		RedisURL:          firstNonEmpty(v.GetString("redis_url"), os.Getenv("REDIS_URL")),
		MasterKey:         v.GetString("master_key"),
		JWTSecret:         firstNonEmpty(v.GetString("jwt_secret"), v.GetString("master_key")),
		YYDSAPIKey:        v.GetString("yyds_api_key"),
		YYDSDomain:        v.GetString("yyds_domain"),
		LogLevel:          v.GetString("log_level"),
		ReadyRequireRedis: v.GetBool("ready_require_redis"),
	}

	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":17890"
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL / GROKFORGE_DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL / GROKFORGE_REDIS_URL is required")
	}
	if len(cfg.JWTSecret) < 16 {
		return nil, fmt.Errorf("GROKFORGE_JWT_SECRET (or GROKFORGE_MASTER_KEY fallback) must be at least 16 characters")
	}
	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Redacted returns a copy safe for logs (secrets stripped).
func (c *Config) Redacted() map[string]any {
	return map[string]any{
		"http_addr":           c.HTTPAddr,
		"database_url_set":    c.DatabaseURL != "",
		"redis_url_set":       c.RedisURL != "",
		"master_key_set":      c.MasterKey != "",
		"jwt_secret_set":      c.JWTSecret != "",
		"yyds_api_key_set":    c.YYDSAPIKey != "",
		"yyds_domain":         c.YYDSDomain,
		"log_level":           c.LogLevel,
		"ready_require_redis": c.ReadyRequireRedis,
	}
}

// Default timeouts used by infrastructure clients.
const (
	DefaultDBPingTimeout    = 3 * time.Second
	DefaultRedisPingTimeout = 2 * time.Second
)
