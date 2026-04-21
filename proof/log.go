package proof

import (
	"context"
	"io"
	"log/slog"
)

type loggerContextKey struct{}

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

func loggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return discardLogger
	}
	logger, ok := ctx.Value(loggerContextKey{}).(*slog.Logger)
	if !ok || logger == nil {
		return discardLogger
	}
	return logger
}
