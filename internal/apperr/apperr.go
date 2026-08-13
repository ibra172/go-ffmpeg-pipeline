package apperr

import "errors"

var (
	// ErrNotFound — запрашиваемая сущность не найдена (HTTP 404).
	ErrNotFound = errors.New("not found")

	// ErrInvalidArgument — переданы некорректные данные (HTTP 400).
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrConflict — конфликт при обновлении (HTTP 409)
	ErrConflict = errors.New("conflict")

	ErrUnauthorized = errors.New("unauthorized")
)
