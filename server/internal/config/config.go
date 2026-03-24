package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all server configuration loaded from environment variables.
// Every field maps 1:1 to an env var documented in the comment.
type Config struct {
	// Server
	Port            string        // PORT (default "8080")
	ShutdownTimeout time.Duration // SHUTDOWN_TIMEOUT (default 25s)
	LogLevel        string        // LOG_LEVEL (default "info")

	// Database
	DatabaseURL         string        // DATABASE_URL (required)
	DBMaxConns          int           // DB_MAX_CONNS (default 25)
	DBMinConns          int           // DB_MIN_CONNS (default 5)
	DBMaxConnLifetime   time.Duration // DB_MAX_CONN_LIFETIME (default 5m)
	DBHealthCheckPeriod time.Duration // DB_HEALTH_CHECK_PERIOD (default 30s)

	// S3
	S3Endpoint  string // S3_ENDPOINT (required)
	S3Bucket    string // S3_BUCKET (default "screenspace")
	S3AccessKey string // S3_ACCESS_KEY (required)
	S3SecretKey string // S3_SECRET_KEY (required)

	// Auth
	JWTSecret      string        // JWT_SECRET (required, min 32 chars)
	JWTExpiry      time.Duration // JWT_EXPIRY (default 168h = 7d)
	AdminEmail     string        // ADMIN_EMAIL
	BcryptCost     int           // BCRYPT_COST (default 10)
	MinPasswordLen int           // MIN_PASSWORD_LENGTH (default 8)

	// Rate Limits
	AuthRateLimit     int // AUTH_RATE_LIMIT (default 10)
	PublicRateLimit   int // PUBLIC_RATE_LIMIT (default 120)
	UserRateLimit     int // USER_RATE_LIMIT (default 30)
	UploadRateLimit   int // UPLOAD_RATE_LIMIT (default 5)
	DownloadRateLimit int // DOWNLOAD_RATE_LIMIT (default 60)

	// Upload Constraints
	MaxFileSize     int64   // MAX_FILE_SIZE (default 209715200 = 200MB)
	MaxDuration     float64 // MAX_DURATION (default 60.0)
	MinHeight       int     // MIN_HEIGHT (default 1080)
	MaxTitleLength  int     // MAX_TITLE_LENGTH (default 255)
	MaxTagCount     int     // MAX_TAG_COUNT (default 10)
	MaxTagLength    int     // MAX_TAG_LENGTH (default 50)
	MaxReportLength int     // MAX_REPORT_LENGTH (default 500)

	// Pagination
	DefaultPageSize int // DEFAULT_PAGE_SIZE (default 20)
	MaxPageSize     int // MAX_PAGE_SIZE (default 100)

	// Presigned URL Expiry
	PresignedDownloadExpiry time.Duration // PRESIGNED_DOWNLOAD_EXPIRY (default 1h)
	PresignedUploadExpiry   time.Duration // PRESIGNED_UPLOAD_EXPIRY (default 2h)
}

// Load reads configuration from environment variables and returns a validated Config.
// Missing required fields or invalid values cause an error.
func Load() (*Config, error) {
	cfg := &Config{
		Port:            envStr("PORT", "8080"),
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT", 25*time.Second),
		LogLevel:        envStr("LOG_LEVEL", "info"),

		DatabaseURL:         os.Getenv("DATABASE_URL"),
		DBMaxConns:          envInt("DB_MAX_CONNS", 25),
		DBMinConns:          envInt("DB_MIN_CONNS", 5),
		DBMaxConnLifetime:   envDuration("DB_MAX_CONN_LIFETIME", 5*time.Minute),
		DBHealthCheckPeriod: envDuration("DB_HEALTH_CHECK_PERIOD", 30*time.Second),

		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3Bucket:    envStr("S3_BUCKET", "screenspace"),
		S3AccessKey: os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey: os.Getenv("S3_SECRET_KEY"),

		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTExpiry:      envDuration("JWT_EXPIRY", 7*24*time.Hour),
		AdminEmail:     os.Getenv("ADMIN_EMAIL"),
		BcryptCost:     envInt("BCRYPT_COST", 10),
		MinPasswordLen: envInt("MIN_PASSWORD_LENGTH", 8),

		AuthRateLimit:     envInt("AUTH_RATE_LIMIT", 10),
		PublicRateLimit:   envInt("PUBLIC_RATE_LIMIT", 120),
		UserRateLimit:     envInt("USER_RATE_LIMIT", 30),
		UploadRateLimit:   envInt("UPLOAD_RATE_LIMIT", 5),
		DownloadRateLimit: envInt("DOWNLOAD_RATE_LIMIT", 60),

		MaxFileSize:     envInt64("MAX_FILE_SIZE", 200*1024*1024),
		MaxDuration:     envFloat64("MAX_DURATION", 60.0),
		MinHeight:       envInt("MIN_HEIGHT", 1080),
		MaxTitleLength:  envInt("MAX_TITLE_LENGTH", 255),
		MaxTagCount:     envInt("MAX_TAG_COUNT", 10),
		MaxTagLength:    envInt("MAX_TAG_LENGTH", 50),
		MaxReportLength: envInt("MAX_REPORT_LENGTH", 500),

		DefaultPageSize: envInt("DEFAULT_PAGE_SIZE", 20),
		MaxPageSize:     envInt("MAX_PAGE_SIZE", 100),

		PresignedDownloadExpiry: envDuration("PRESIGNED_DOWNLOAD_EXPIRY", time.Hour),
		PresignedUploadExpiry:   envDuration("PRESIGNED_UPLOAD_EXPIRY", 2*time.Hour),
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	var errs []error

	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.JWTSecret == "" {
		errs = append(errs, errors.New("JWT_SECRET is required"))
	} else if len(c.JWTSecret) < 32 {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d)", len(c.JWTSecret)))
	}
	if c.S3Endpoint == "" {
		errs = append(errs, errors.New("S3_ENDPOINT is required"))
	}
	if c.S3AccessKey == "" {
		errs = append(errs, errors.New("S3_ACCESS_KEY is required"))
	}
	if c.S3SecretKey == "" {
		errs = append(errs, errors.New("S3_SECRET_KEY is required"))
	}
	if c.DBMaxConns < 1 {
		errs = append(errs, errors.New("DB_MAX_CONNS must be >= 1"))
	}
	if c.DBMinConns < 0 {
		errs = append(errs, errors.New("DB_MIN_CONNS must be >= 0"))
	}
	if c.DBMinConns > c.DBMaxConns {
		errs = append(errs, errors.New("DB_MIN_CONNS must be <= DB_MAX_CONNS"))
	}
	if c.MaxPageSize < c.DefaultPageSize {
		errs = append(errs, errors.New("MAX_PAGE_SIZE must be >= DEFAULT_PAGE_SIZE"))
	}

	return errors.Join(errs...)
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func envFloat64(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
