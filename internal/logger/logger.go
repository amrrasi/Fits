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

type Config struct {
	Level       string
	LogDir      string
	AppName     string
	Development bool
}

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

	if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
		return fmt.Errorf("logger: create log dir: %w", err)
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return fmt.Errorf("logger: invalid level %q: %w", cfg.Level, err)
	}

	today := time.Now().Format("2006-01-02")
	logPath := filepath.Join(cfg.LogDir, fmt.Sprintf("%s-%s.log", cfg.AppName, today))
	fileSink, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("logger: open log file: %w", err)
	}

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
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(fileSink), level),
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(errSink), zap.ErrorLevel),
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

func L() *zap.Logger {
	if global == nil {
		panic("logger: Init() not called")
	}
	return global
}

func S() *zap.SugaredLogger {
	return L().Sugar()
}

func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}
