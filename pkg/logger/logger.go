package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger interface that both services can implement
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	With(fields ...zap.Field) Logger
	Sync() error
}

// zapLogger wraps zap.Logger to implement our Logger interface
type zapLogger struct {
	*zap.Logger
}

// Config holds logger configuration
type Config struct {
	Level      string `json:"level" yaml:"level"`            // debug, info, warn, error
	Format     string `json:"format" yaml:"format"`          // json or console
	OutputPath string `json:"output_path" yaml:"outputPath"` // stdout or file path
}

type Environment string

const (
	Dev  Environment = "dev"
	Prod Environment = "prod"
)

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "json",
		OutputPath: "stdout",
	}
}

// New creates a new Logger with the given configuration
func New(cfg *Config) (Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create zap configuration
	zapConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(level),
		Development:      false,
		Encoding:         cfg.Format,
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{cfg.OutputPath},
		ErrorOutputPaths: []string{cfg.OutputPath},
	}

	// Build the logger
	logger, err := zapConfig.Build(
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return &zapLogger{logger}, nil
}

// Ensure zapLogger implements Logger interface
var _ Logger = (*zapLogger)(nil)

// With creates a child logger with the given fields
func (l *zapLogger) With(fields ...zap.Field) Logger {
	return &zapLogger{l.Logger.With(fields...)}
}

// Example function to create a development logger quickly
func NewDevelopment() (Logger, error) {
	cfg := &Config{
		Level:      "debug",
		Format:     "console",
		OutputPath: "stdout",
	}
	return New(cfg)
}

// Example function to create a production logger quickly
func NewProduction() (Logger, error) {
	cfg := &Config{
		Level:      "info",
		Format:     "json",
		OutputPath: "stdout",
	}
	return New(cfg)
}

func SetupLogger(env Environment) (Logger, error) {
	switch env {
	case Dev:
		return New(&Config{
			Level:      "debug",
			Format:     "console",
			OutputPath: "stdout",
		})
	case Prod:
		return New(&Config{
			Level:      "info",
			Format:     "json",
			OutputPath: "/var/log/app.log", // or use stdout for container environments
		})
	default:
		return NewDevelopment()
	}
}
