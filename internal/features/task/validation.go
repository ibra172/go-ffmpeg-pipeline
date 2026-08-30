package task

import (
	"fmt"
	"slices"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
)

var (
	allowedOperations  = []string{"transcode", "thumbnail", "watermark"}
	allowedFormats     = []string{"mp4", "mov", "mkv", "webm"}
	allowedResolutions = []string{"360p", "480p", "720p", "1080p", "4k"}
)

func validateCreateTaskRequest(req CreateTaskRequest) error {
	if !slices.Contains(allowedOperations, req.Operation) {
		return fmt.Errorf(
			"invalid operation %q, allowed: %v: %w",
			req.Operation, allowedOperations, apperr.ErrInvalidArgument)
	}

	if req.Operation == "transcode" {
		if !slices.Contains(allowedFormats, req.TargetFormat) {
			return fmt.Errorf("invalid target_format %q, allowed: %v: %w",
				req.TargetFormat, allowedFormats, apperr.ErrInvalidArgument)
		}
		if req.Resolution != "" && !slices.Contains(allowedResolutions, req.Resolution) {
			return fmt.Errorf("invalid resolution %q, allowed: %v: %w",
				req.Resolution, allowedResolutions, apperr.ErrInvalidArgument)
		}
	}

	return nil
}
