// Package logger provides a structured, leveled logger for the FITS processor.
// It writes to both stdout and a rotating log file under the logs/ directory.
// Usage: call logger.Init() once at startup, then use logger.L() anywhere.
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

// Config controls how the logger is initialised.
type Config struct {
	// Level is the minimum log level: "debug", "info", "warn", "error".
	Level string
	// LogDir is the directory where log files are written. Defaults to "logs/".
	LogDir string
	// AppName is used as the log file prefix.
	AppName string
	// Development enables caller info and stack traces on Warn+.
	Development bool
}

// Init initialises the global logger.  Call once from main() before anything else.
func Init(cfg Config) error {
	if cfg.LogDir == "" {
		cfg.LogDir = "logs"
	}
	if cfg.AppName == "" {
		cfg.AppName = "fits-processor"
	}
	if cfg.Level == "" {
		cfg.Level = "info"
	}

	// Ensure log directory exists
	if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
		return fmt.Errorf("logger: create log dir: %w", err)
	}

	// Parse level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return fmt.Errorf("logger: invalid level %q: %w", cfg.Level, err)
	}

	// File sink — daily rotating name: appname-2006-01-02.log
	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(cfg.LogDir, fmt.Sprintf("%s-%s.log", cfg.AppName, today))
	fileSink, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("logger: open log file: %w", err)
	}

	// Error file sink — separate file for errors only
	errPath := filepath.Join(cfg.LogDir, fmt.Sprintf("%s-%s.error.log", cfg.AppName, today))
	errSink, err := os.OpenFile(errPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("logger: open error log file: %w", err)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	jsonEncoder := zapcore.NewJSONEncoder(encoderCfg)
	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)

	core := zapcore.NewTee(
		// JSON to file (all levels)
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(fileSink), level),
		// JSON errors-only to error file
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(errSink), zap.ErrorLevel),
		// Human-readable to stdout
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
	)

	opts := []zap.Option{zap.AddCaller()}
	if cfg.Development {
		opts = append(opts, zap.Development(), zap.AddStacktrace(zap.WarnLevel))
	} else {
		opts = append(opts, zap.AddStacktrace(zap.ErrorLevel))
	}

	global = zap.New(core, opts...)
	return nil
}

// L returns the global logger. Panics if Init has not been called.
func L() *zap.Logger {
	if global == nil {
		panic("logger: Init() not called")
	}
	return global
}

// S returns the global sugared logger (printf-style). Panics if Init has not been called.
func S() *zap.SugaredLogger {
	return L().Sugar()
}

// Sync flushes any buffered log entries. Call defer logger.Sync() in main().
func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}
