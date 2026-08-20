package auth

import (
	"context"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSessionByToken(ctx context.Context, token string) (Session, error)
}
