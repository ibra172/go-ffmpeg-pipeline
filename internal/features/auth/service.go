package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, username string, password string) (*User, error)
	Login(ctx context.Context, username string, password string) (*Session, error)
	Authenticate(ctx context.Context, bearerToken string) (*User, error)
}

type AuthService struct {
	UserRepository    UserRepository    // интерфейс
	SessionRepository SessionRepository // интерфейс
}

func NewService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
) *AuthService {
	return &AuthService{
		UserRepository:    userRepo,
		SessionRepository: sessionRepo,
	}
}

var (
	cost = bcrypt.DefaultCost

	// Фиктивный хеш, сгенерированный один раз при старте приложения
	dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy_password"), cost)
)

func (s *AuthService) Register(ctx context.Context, username string, password string) (*User, error) {
	_, err := s.UserRepository.GetUserByUsername(ctx, username)
	if err == nil {
		return &User{}, fmt.Errorf("the username `%s` is already taken: %w", username, apperr.ErrConflict)
	}
	if !errors.Is(err, apperr.ErrNotFound) {
		return &User{}, fmt.Errorf("check username: %w", err)
	}

	pswHash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return &User{}, fmt.Errorf("generate password hash: %w", err)
	}

	user := &User{
		ID:           uuid.New(),
		Username:     username,
		PasswordHash: string(pswHash),
	}
	if err = s.UserRepository.CreateUser(ctx, user); err != nil {
		return &User{}, fmt.Errorf("save user in repository: %w", err)
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, username string, password string) (*Session, error) {
	// найти пользователя
	user, err := s.UserRepository.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			// Симулирование проверки, чтобы не выдать отсутствие пользователя по времени
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return &Session{}, fmt.Errorf("incorrect username or password: %w", apperr.ErrUnauthorized)
		}

		return &Session{}, fmt.Errorf("get user from repository: %w", err)
	}

	// сверить хеш
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return &Session{}, fmt.Errorf("incorrect username or password: %w", apperr.ErrUnauthorized)
	}

	// создать сессию, сохранить и вернуть
	session := &Session{
		Token:     uuid.NewString(),
		UserID:    user.ID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := s.SessionRepository.CreateSession(ctx, session); err != nil {
		return &Session{}, fmt.Errorf("save session in repository: %w", err)
	}

	return session, nil
}

func (s *AuthService) Authenticate(ctx context.Context, bearerToken string) (*User, error) {
	session, err := s.SessionRepository.GetSessionByToken(ctx, bearerToken)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return &User{}, fmt.Errorf("invalid token: %w", apperr.ErrUnauthorized)
		}
		return &User{}, fmt.Errorf("get session from repository: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		return &User{}, fmt.Errorf("token expired: %w", apperr.ErrUnauthorized)
	}

	user, err := s.UserRepository.GetUserByID(ctx, session.UserID)
	if err != nil {
		return &User{}, fmt.Errorf("get user from repository: %w", err)
	}

	return &user, nil
}
