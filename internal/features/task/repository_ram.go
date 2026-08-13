package task

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
)

type RamRepository struct {
	data map[uuid.UUID]*Task
	mu   sync.RWMutex
}

func NewRamRepository() *RamRepository {
	return &RamRepository{
		data: make(map[uuid.UUID]*Task),
	}
}

func (r *RamRepository) CreateTask(ctx context.Context, task *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[task.ID] = task

	return nil
}

func (r *RamRepository) GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.data[id]
	if !ok {
		return Task{}, fmt.Errorf("task with ID=`%s`: %w", id, apperr.ErrNotFound)
	}

	return *task, nil
}

func (r *RamRepository) UpdateTask(ctx context.Context, task *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[task.ID]; !exists {
		return fmt.Errorf("task with ID=`%s`: %w", task.ID, apperr.ErrNotFound)
	}

	r.data[task.ID] = task

	return nil
}
