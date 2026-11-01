package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
)

func TestValidateRegistration(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{"valid credentials", "Alice_1", "12345678", false},
		{"empty username", "", "12345678", true},
		{"username too short", "Ab", "12345678", true},
		{"username at maximum length", strings.Repeat("a", 30), "12345678", false},
		{"username too long", strings.Repeat("a", 31), "12345678", true},
		{"username starts with digit", "1_May", "12345678", true},
		{"password too short", "alice", "1234567", true},
		{"password at maximum length", "alice", "ThisIs72Bytes!ThisIs72Bytes!ThisIs72Bytes!ThisIs72Bytes!ThisIs72Bytes!ok", false},
		{"password too long", "alice", "ThisIs73Bytes!ThisIs73Bytes!ThisIs73Bytes!ThisIs73Bytes!ThisIs73Bytes!ok!", true},
		{"multibyte password at maximum length", "alice", strings.Repeat("я", 36), false},
		{"multibyte password too long", "alice", strings.Repeat("я", 37), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRegistration(tc.username, tc.password)
			if (err != nil) != tc.wantErr {
				t.Fatalf(
					"validateRegistration() error = %v, wantErr = %t",
					err, tc.wantErr,
				)
			}
			if err != nil {
				if !errors.Is(err, apperr.ErrInvalidArgument) {
					t.Fatalf("expected ErrInvalidArgument, got %v", err)
				}
				var validationErr *validationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("expected validationError, got %v", err)
				}
			}
		})
	}
}
