package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DB   DBConfig
	Log  LogConfig
	FITS FITSConfig
	App  AppConfig
	JWT  JWTConfig
	HTTP HTTPConfig
}

type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

func (d DBConfig) PgxDSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:     "/" + d.Name,
		RawQuery: "sslmode=" + url.QueryEscape(d.SSLMode),
	}
	return u.String()
}

type LogConfig struct {
	Level       string
	Dir         string
	Development bool
}

type FITSConfig struct {
	ScanDir   string
	Workers   int
	BatchSize int
}

type AppConfig struct {
	Name          string
	Environment   string
	MigrationsDir string
}

type JWTConfig struct {
	AccessSecret string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
}

type HTTPConfig struct {
	Addr string

	CORSAllowedOrigins []string

	RateLimitAuthRPS   float64
	RateLimitAuthBurst float64
	RateLimitAPIRPS    float64
	RateLimitAPIBurst  float64

	MaxBodyBytes int64

	StaticDir string

	TrustedProxies []string
	CookieSecure   bool
	AdminEmail     string
	AdminPassword  string
}

var insecureDefaults = []string{
	"change-me-access-secret-32chars!!",
	"change-me-refresh-secret-32chars!",
	"change-me-access-secret-min-32-chars!!",
	"change-me-refresh-secret-min-32-chars!",
}

func Load(envFile string) (*Config, error) {
	if envFile != "" {
		_ = godotenv.Load(envFile)
	}

	cfg := &Config{}

	cfg.DB = DBConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "amirpopass83"),
		Name:            getEnv("DB_NAME", "fits_db"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}

	cfg.Log = LogConfig{
		Level:       getEnv("LOG_LEVEL", "info"),
		Dir:         getEnv("LOG_DIR", "logs"),
		Development: getEnvBool("LOG_DEVELOPMENT", false),
	}

	cfg.FITS = FITSConfig{
		ScanDir:   getEnv("FITS_SCAN_DIR", "./testdata"),
		Workers:   getEnvInt("FITS_WORKERS", 4),
		BatchSize: getEnvInt("FITS_BATCH_SIZE", 100),
	}

	cfg.JWT = JWTConfig{
		AccessSecret: getEnv("JWT_ACCESS_SECRET", "change-me-access-secret-32chars!!"),
		AccessTTL:    getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:   getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
	}

	cfg.App = AppConfig{
		Name:          getEnv("APP_NAME", "fits-processor"),
		Environment:   getEnv("APP_ENV", "development"),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "migrations"),
	}

	rawOrigins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	var origins []string
	for _, o := range strings.Split(rawOrigins, ",") {
		if s := strings.TrimSpace(o); s != "" {
			origins = append(origins, s)
		}
	}

	cfg.HTTP = HTTPConfig{
		Addr:               getEnv("SERVER_ADDR", ":8080"),
		CORSAllowedOrigins: origins,
		RateLimitAuthRPS:   getEnvFloat("RATE_LIMIT_AUTH_RPS", 5),
		RateLimitAuthBurst: getEnvFloat("RATE_LIMIT_AUTH_BURST", 10),
		RateLimitAPIRPS:    getEnvFloat("RATE_LIMIT_API_RPS", 60),
		RateLimitAPIBurst:  getEnvFloat("RATE_LIMIT_API_BURST", 120),
		MaxBodyBytes:       int64(getEnvInt("MAX_BODY_BYTES", 1<<20)), // 1 MB
		StaticDir:          getEnv("STATIC_DIR", ""),
		TrustedProxies:     splitCSV(getEnv("TRUSTED_PROXIES", "")),
		CookieSecure:       getEnvBool("COOKIE_SECURE", getEnv("APP_ENV", "development") == "production"),
		AdminEmail:         strings.ToLower(strings.TrimSpace(getEnv("ADMIN_EMAIL", "admin@fits.local"))),
		AdminPassword:      getEnv("ADMIN_PASSWORD", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	isProd := c.App.Environment == "production"

	if c.DB.Password == "" && isProd {
		return fmt.Errorf("DB_PASSWORD must be set in production")
	}
	if c.FITS.Workers < 1 {
		return fmt.Errorf("FITS_WORKERS must be >= 1")
	}
	if c.FITS.BatchSize < 1 {
		return fmt.Errorf("FITS_BATCH_SIZE must be >= 1")
	}

	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET must not be empty")
	}
	if isProd {
		for _, o := range c.HTTP.CORSAllowedOrigins {
			if o == "*" {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain * in production")
			}
		}
		for _, bad := range insecureDefaults {
			if c.JWT.AccessSecret == bad {
				return fmt.Errorf("JWT_ACCESS_SECRET must be changed from the default in production")
			}
		}
		if len(c.JWT.AccessSecret) < 32 {
			return fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters in production")
		}
	}

	return nil
}

func splitCSV(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v, ok := os.LookupEnv(key); ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
