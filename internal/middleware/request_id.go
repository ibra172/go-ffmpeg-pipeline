package middleware

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/ctxlog"
)

const requestIDHeader = "X-Request-ID"

func RequestID(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = uuid.NewString()
			}
			w.Header().Set(requestIDHeader, requestID)

			logger := baseLogger.With("request_id", requestID)
			ctx := ctxlog.ToContext(r.Context(), logger)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
