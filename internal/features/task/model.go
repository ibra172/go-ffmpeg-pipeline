package task

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusReady      Status = "ready"
	StatusError      Status = "error"
)

type Payload struct {
	Operation    string
	TargetFormat string
	Resolution   string
	InputPath    string
}

type Result struct {
	OutputPath string
}

// Task — доменная модель задачи.
type Task struct {
	ID      uuid.UUID
	Status  Status
	Payload Payload

	Result   *Result
	ErrorMsg string

	CreatedAt time.Time
	UpdatedAt time.Time
}
