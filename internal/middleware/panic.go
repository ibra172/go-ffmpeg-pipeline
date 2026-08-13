package middleware

import (
	"net/http"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/ctxlog"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/httpresp"
)

// Panic — middleware для перехвата паник и возврата HTTP 500.
// Без этого middleware паника в обработчике уронила бы всю горутину,
// а стандартная библиотека Go вернула бы пустой ответ клиенту.
//
// Использует defer + recover — стандартный паттерн обработки паник в Go.
func Panic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				ctxlog.FromContext(r.Context()).Error("panic recovered", "panic", p)
				httpresp.RespondJSON(
					w,
					http.StatusInternalServerError,
					map[string]string{"error": "internal server error"},
				)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
