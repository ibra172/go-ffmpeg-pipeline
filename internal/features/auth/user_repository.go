package auth

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	GetUserByUsername(ctx context.Context, username string) (User, error)
	UpdateUser(ctx context.Context, user *User) error
}
