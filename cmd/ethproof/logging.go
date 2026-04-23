package main

import (
	"log/slog"
	"os"

	"github.com/islishude/ethproof/internal/logutil"
)

type runtimeCommandError struct {
	err    error
	logger *slog.Logger
}

func (e runtimeCommandError) Error() string {
	return e.err.Error()
}

func (e runtimeCommandError) Unwrap() error {
	return e.err
}

func newCommandLogger(cfg logutil.Config) *slog.Logger {
	return logutil.MustNewLogger(os.Stderr, cfg)
}

func wrapRuntimeError(logger *slog.Logger, err error) error {
	if err == nil {
		return nil
	}
	return runtimeCommandError{
		err:    err,
		logger: logger,
	}
}
