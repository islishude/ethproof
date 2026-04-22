package proof

import (
	"context"
	"log/slog"
)

// Deprecated: proof package no longer emits logs. WithLogger returns ctx unchanged.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	_ = logger
	if ctx == nil {
		ctx = context.Background()
	}
	return ctx
}
