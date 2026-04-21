package logutil

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	DefaultLevel  = "info"
	DefaultFormat = "text"
)

type Config struct {
	Level  string
	Format string
}

func DefaultConfig() Config {
	return Config{
		Level:  DefaultLevel,
		Format: DefaultFormat,
	}
}

func NormalizeConfig(cfg Config) (Config, error) {
	out := Config{
		Level:  strings.ToLower(strings.TrimSpace(cfg.Level)),
		Format: strings.ToLower(strings.TrimSpace(cfg.Format)),
	}
	if out.Level == "" {
		out.Level = DefaultLevel
	}
	if out.Format == "" {
		out.Format = DefaultFormat
	}
	if _, err := parseLevel(out.Level); err != nil {
		return Config{}, err
	}
	switch out.Format {
	case "text", "json":
	default:
		return Config{}, fmt.Errorf("unsupported log format %q (want text or json)", cfg.Format)
	}
	return out, nil
}

func NewLogger(w io.Writer, cfg Config) (*slog.Logger, error) {
	normalized, err := NormalizeConfig(cfg)
	if err != nil {
		return nil, err
	}
	level, err := parseLevel(normalized.Level)
	if err != nil {
		return nil, err
	}
	opts := &slog.HandlerOptions{Level: level}
	switch normalized.Format {
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	case "text":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	default:
		return nil, fmt.Errorf("unsupported log format %q", normalized.Format)
	}
}

func MustNewLogger(w io.Writer, cfg Config) *slog.Logger {
	logger, err := NewLogger(w, cfg)
	if err != nil {
		panic(err)
	}
	return logger
}

func parseLevel(raw string) (slog.Level, error) {
	switch raw {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level %q (want debug, info, warn, or error)", raw)
	}
}
