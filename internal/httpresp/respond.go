package httpresp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/ctxlog"
)

// ErrorResponse describes an API error payload.
// swagger:model ErrorResponse
type ErrorResponse struct {
	// Error holds a public error message.
	Error string `json:"error"`
}

func RespondJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func RespondError(ctx context.Context, w http.ResponseWriter, err error, publicMsg string) {
	status := statusFromError(err)

	logger := ctxlog.FromContext(ctx)
	if status >= 500 {
		logger.Error(publicMsg, "error", err, "status", status)
	} else {
		logger.Warn(publicMsg, "error", err, "status", status)
	}

	RespondJSON(w, status, ErrorResponse{Error: publicMsg})
}

func statusFromError(err error) int {
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperr.ErrInvalidArgument):
		return http.StatusBadRequest
	case errors.Is(err, apperr.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, apperr.ErrUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
