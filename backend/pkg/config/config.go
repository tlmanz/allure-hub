// Package config loads application configuration from environment variables
// using goconf + caarlos0/env. Settings are grouped into sub-structs for
// clarity; environment variable names are unchanged from previous releases.
package config

import (
	"errors"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
)

// ServerConfig holds HTTP server tuning parameters.
// Critical for large (100–200 MB) file uploads — ReadTimeout must exceed the
// longest possible upload, and WriteTimeout must cover report serving and SSE.
type ServerConfig struct {
	Addr              string        `env:"ADDR"               envDefault:":8080"`
	MaxHeaderBytes    int           `env:"MAX_HEADER_BYTES"   envDefault:"1048576"` // 1 MB
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"10s"`    // Slowloris protection
	ReadTimeout       time.Duration `env:"READ_TIMEOUT"       envDefault:"2h"`      // must exceed largest upload duration
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT"      envDefault:"10m"`     // covers report serving; SSE clients reconnect on expiry
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT"       envDefault:"120s"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT"   envDefault:"30s"` // graceful drain period
	WebDir            string        `env:"WEB_DIR"            envDefault:"./app/web"`
}

// DBConfig holds database driver, DSN, and connection pool parameters.
// DB_DSN is masked in log output — never printed in plaintext.
// For PostgreSQL use: postgres://user:password@host:5432/dbname?sslmode=require
type DBConfig struct {
	Driver          string        `env:"DB_DRIVER"           envDefault:"sqlite"`
	DSN             string        `env:"DB_DSN"              envDefault:"./data/allure-hub.db" hush:"mask"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS"   envDefault:"25"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS"   envDefault:"5"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"30m"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"5m"`
}

// StorageConfig holds file system paths and upload size limits.
// All size caps are defence-in-depth: MaxDecompressedBytes prevents zip-bomb
// exhaustion (M-03) and MaxZipEntries prevents inode exhaustion (M-04).
type StorageConfig struct {
	DataDir              string `env:"DATA_DIR"              envDefault:"./data"`
	AssembleTempDir      string `env:"ASSEMBLE_TEMP_DIR"     envDefault:"./temp"`      // staging dir for chunk assembly
	MaxChunkBytes        int64  `env:"MAX_CHUNK_BYTES"       envDefault:"52428800"`    // 50 MB per chunk
	MaxUploadBytes       int64  `env:"MAX_UPLOAD_BYTES"      envDefault:"1073741824"`  // 1 GB compressed cap
	MaxDecompressedBytes int64  `env:"MAX_DECOMPRESSED_BYTES" envDefault:"1610612736"` // 1.5 GB decompressed cap (M-03)
	MaxZipEntries        int    `env:"MAX_ZIP_ENTRIES"       envDefault:"10000"`       // max files in a zip (M-04)
}

// AllureConfig holds Allure CLI invocation parameters.
type AllureConfig struct {
	Bin            string        `env:"ALLURE_BIN"             envDefault:"allure"`
	ConfigPath     string        `env:"ALLURE_CONFIG"          envDefault:"./settings/allurerc.yml"`
	MaxConcurrency int           `env:"ALLURE_MAX_CONCURRENCY" envDefault:"4"`
	Timeout        time.Duration `env:"ALLURE_TIMEOUT"         envDefault:"10m"` // per-invocation deadline
}

// RateLimitConfig holds per-IP token bucket parameters for the /api/ routes.
// Set Rate ≤ 0 to disable. TrustProxy enables X-Forwarded-For header parsing.
type RateLimitConfig struct {
	Rate       float64 `env:"RATE_LIMIT_RATE"  envDefault:"20"`    // tokens per second
	Burst      float64 `env:"RATE_LIMIT_BURST" envDefault:"60"`    // burst capacity
	TrustProxy bool    `env:"TRUST_PROXY"      envDefault:"false"` // trust X-Forwarded-For
}

// CORSConfig holds allowed origins for cross-origin API requests.
// Comma-separated list; use "*" in development only. Empty = same-origin only.
type CORSConfig struct {
	AllowedOrigins string `env:"CORS_ALLOWED_ORIGINS" envDefault:""`
}

// LogConfig holds zap logger parameters.
type LogConfig struct {
	Level  string `env:"LOG_LEVEL"  envDefault:"info"`   // debug | info | warn | error
	Format string `env:"LOG_FORMAT" envDefault:"json"`   // json | console
	Output string `env:"LOG_OUTPUT" envDefault:"stdout"` // stdout | stderr
}

// AuthConfig holds authkit (OAuth + session + RBAC) parameters.
type AuthConfig struct {
	SessionSecret      string `env:"SESSION_SECRET,required"  hush:"mask"`
	BaseURL            string `env:"BASE_URL"`
	SecureCookie       bool   `env:"SECURE_COOKIE"            envDefault:"false"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"         hush:"mask"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"     hush:"mask"`
	PolicyFile         string `env:"AUTH_POLICY_FILE"         envDefault:"./policy.yaml"`
	AfterLoginURL      string `env:"AUTH_AFTER_LOGIN_URL"     envDefault:"/"`
	AfterLogoutURL     string `env:"AUTH_AFTER_LOGOUT_URL"    envDefault:"/login"`
}

// CleanupConfig holds startup seed values for the background report cleanup worker.
// All three values are seeded into the system_settings DB table on first run and
// can be overridden at runtime via the Settings API without a server restart.
// Set CLEANUP_INTERVAL to 0 to disable the worker entirely.
type CleanupConfig struct {
	RetentionDays   int           `env:"CLEANUP_RETENTION_DAYS"    envDefault:"90"`
	Interval        time.Duration `env:"CLEANUP_INTERVAL"          envDefault:"6h"`
	DryRun          bool          `env:"CLEANUP_DRY_RUN"           envDefault:"false"`
}

// Config is the root configuration for allure-hub, assembled from environment
// variables. Sub-structs group related settings; env var names are unchanged.
type Config struct {
	Server    ServerConfig
	DB        DBConfig
	Storage   StorageConfig
	Allure    AllureConfig
	RateLimit RateLimitConfig
	CORS      CORSConfig
	Log       LogConfig
	Auth      AuthConfig
	Cleanup   CleanupConfig
}

// Values is the package-level instance populated by Load().
var Values Config

// Register parses environment variables into Values, satisfying goconf.Configer.
func (Config) Register() error {
	return env.Parse(&Values)
}

// Validate checks required field constraints after parsing, satisfying goconf.Validater.
func (Config) Validate() error {
	if Values.Storage.DataDir == "" {
		return errors.New("DATA_DIR must not be empty")
	}
	if Values.DB.Driver != "sqlite" && Values.DB.Driver != "postgres" {
		return errors.New("DB_DRIVER must be \"sqlite\" or \"postgres\"")
	}
	switch strings.ToLower(Values.Log.Level) {
	case "debug", "info", "warn", "error":
	default:
		return errors.New("LOG_LEVEL must be one of: debug, info, warn, error")
	}
	switch strings.ToLower(Values.Log.Format) {
	case "json", "console":
	default:
		return errors.New("LOG_FORMAT must be one of: json, console")
	}
	switch strings.ToLower(Values.Log.Output) {
	case "stdout", "stderr":
	default:
		return errors.New("LOG_OUTPUT must be one of: stdout, stderr")
	}
	return nil
}

// Print returns the config struct for goconf's table printer, satisfying goconf.Printer.
func (Config) Print() any {
	return Values
}
