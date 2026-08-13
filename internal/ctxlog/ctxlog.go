package ctxlog

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(ctxKey{}).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return logger
}
