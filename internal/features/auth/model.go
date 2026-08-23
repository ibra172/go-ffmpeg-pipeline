package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
}

type Session struct {
	Token     string
	UserID    uuid.UUID
	CreatedAt time.Time
	ExpiresAt time.Time
}
