package ctxlog

import (
	"context"
	"log/slog"
)

type loggerCtxKey struct{}

var key = loggerCtxKey{}

func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, key, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(key).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return logger
}
