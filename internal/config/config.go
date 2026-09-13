package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DB   DBConfig
	Log  LogConfig
	FITS FITSConfig
	App  AppConfig
	JWT  JWTConfig
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string

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
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
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

func Load(envFile string) (*Config, error) {
	if envFile != "" {
		_ = godotenv.Load(envFile)
	}

	cfg := &Config{}

	cfg.DB = DBConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", ""),
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
		AccessSecret:  getEnv("JWT_ACCESS_SECRET", "change-me-access-secret-32chars!!"),
		RefreshSecret: getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret-32chars!"),
		AccessTTL:     getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
	}

	cfg.App = AppConfig{
		Name:          getEnv("APP_NAME", "fits-processor"),
		Environment:   getEnv("APP_ENV", "development"),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "migrations"),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.DB.Password == "" && c.App.Environment == "production" {
		return fmt.Errorf("DB_PASSWORD must be set in production")
	}
	if c.FITS.Workers < 1 {
		return fmt.Errorf("FITS_WORKERS must be >= 1")
	}
	if c.FITS.BatchSize < 1 {
		return fmt.Errorf("FITS_BATCH_SIZE must be >= 1")
	}
	return nil
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
