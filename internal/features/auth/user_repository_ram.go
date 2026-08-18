package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
)

type UserRamRepository struct {
	data map[uuid.UUID]*User
	mu   sync.RWMutex
}

func NewUserRamRepository() *UserRamRepository {
	return &UserRamRepository{
		data: make(map[uuid.UUID]*User),
	}
}

func (r *UserRamRepository) CreateUser(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[user.ID] = user

	return nil
}

func (r *UserRamRepository) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.data[id]
	if !ok {
		return User{}, fmt.Errorf("user with ID=`%v`: %w", id, apperr.ErrNotFound)
	}

	return *user, nil
}

func (r *UserRamRepository) GetUserByUsername(ctx context.Context, username string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.data {
		if user.Username == username {
			return *user, nil
		}
	}

	return User{}, fmt.Errorf("user with usernmame=`%s`: %w", username, apperr.ErrNotFound)
}

func (r *UserRamRepository) UpdateUser(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[user.ID]; !exists {
		return fmt.Errorf("user with ID=`%v`: %w", user.ID, apperr.ErrNotFound)
	}

	r.data[user.ID] = user

	return nil
}
