package middleware

import (
	"net/http"
	"time"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/ctxlog"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/httpresp"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := httpresp.NewResponseWriter(w)

		next.ServeHTTP(rw, r)

		logger := ctxlog.FromContext(r.Context())

		logger.Info(
			"http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.GetStatusCode(),
			"duration", time.Since(start),
		)
	})
}
