package task

import (
	"context"

	"github.com/google/uuid"
)

type Message struct {
	TaskID       uuid.UUID
	Operation    string
	TargetFormat string
	Resolution   string
	InputPath    string
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}
