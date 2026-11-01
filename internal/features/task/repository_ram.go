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
		return Task{}, fmt.Errorf("task with ID=`%v`: %w", id, apperr.ErrNotFound)
	}

	return *task, nil
}

func (r *RamRepository) UpdateTask(ctx context.Context, task *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[task.ID]; !exists {
		return fmt.Errorf("task with ID=`%v`: %w", task.ID, apperr.ErrNotFound)
	}

	if r.data[task.ID].Status != StatusInProgress {
		return fmt.Errorf("task cannot be updated: %w", apperr.ErrConflict)
	}

	r.data[task.ID].Status = task.Status
	r.data[task.ID].Result = task.Result
	r.data[task.ID].ErrorMsg = task.ErrorMsg
	r.data[task.ID].UpdatedAt = task.UpdatedAt

	return nil
}
