package task

import (
	"context"

	"github.com/google/uuid"
)

type TaskRepository interface {
	CreateTask(ctx context.Context, task *Task) error
	GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error)
	UpdateTask(ctx context.Context, task *Task) error
}
