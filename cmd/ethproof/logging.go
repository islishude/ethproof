package main

import (
	"errors"
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

func loggerForError(err error) *slog.Logger {
	var runtimeErr runtimeCommandError
	if errors.As(err, &runtimeErr) && runtimeErr.logger != nil {
		return runtimeErr.logger
	}
	return logutil.MustNewLogger(os.Stderr, logutil.DefaultConfig())
}
