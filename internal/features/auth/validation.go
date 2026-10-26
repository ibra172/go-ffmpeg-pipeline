package auth

import (
	"regexp"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
)

var rxUsername = regexp.MustCompile(`\A[a-zA-Z][a-zA-Z0-9_]{2,29}\z`)

func validateRegistration(username, password string) error {
	if !rxUsername.MatchString(username) {
		return &validationError{
			message: "username must contain from 3 to 30 characters and may consist of Latin letters, numbers, or the '_' character and begin with a letter",
		}
	}

	pswLen := len(password)
	if pswLen < 8 || pswLen > 72 {
		return &validationError{
			message: "password must contain 8 to 72 bytes",
		}
	}

	return nil
}

type validationError struct {
	message string
}

func (e *validationError) Error() string {
	return e.message
}

func (e *validationError) Unwrap() error {
	return apperr.ErrInvalidArgument
}
