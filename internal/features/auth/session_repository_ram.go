package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
)

type SessionRamRepo struct {
	data map[string]*Session
	mu   sync.RWMutex
}

func NewSessionRamRepo() *SessionRamRepo {
	return &SessionRamRepo{
		data: make(map[string]*Session),
	}
}

func (r *SessionRamRepo) CreateSession(ctx context.Context, session *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[session.Token] = session

	return nil
}

func (r *SessionRamRepo) GetSessionByToken(ctx context.Context, token string) (Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.data[token]
	if !ok {
		return Session{}, fmt.Errorf("session with token=`%s`: %w", token, apperr.ErrNotFound)
	}

	return *session, nil
}
