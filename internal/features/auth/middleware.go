package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/httpresp"
)

type userContextKey struct{}

var key = userContextKey{}

func UserToContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, key, user)
}

func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(key).(*User)
	return user, ok
}

func RequireAuth(service Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			authHeader := r.Header.Get("Authorization")

			bearerToken, ok := strings.CutPrefix(authHeader, "Bearer ")
			if !ok || bearerToken == "" {
				httpresp.RespondError(ctx, w, apperr.ErrUnauthorized, "missing or invalid authorization header")
				return
			}

			user, err := service.Authenticate(ctx, bearerToken)
			if err != nil {
				httpresp.RespondError(ctx, w, err, "unauthorized")
				return
			}

			next.ServeHTTP(w, r.WithContext(UserToContext(ctx, user)))
		})
	}
}
