package task

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	StatusInProgress TaskStatus = "in_progress"
	StatusReady      TaskStatus = "ready"
	StatusError      TaskStatus = "error"
)

type TaskPayload struct {
	Operation    string
	TargetFormat string
	Resolution   string
}

type TaskResult struct {
	OutputPath string
}

// Task — доменная модель задачи.
type Task struct {
	ID      uuid.UUID
	Status  TaskStatus
	Payload TaskPayload

	Result   *TaskResult
	ErrorMsg string

	CreatedAt time.Time
	UpdatedAt time.Time
}
