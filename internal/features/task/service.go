package task

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/ctxlog"
)

type TaskService interface {
	CreateTask(ctx context.Context, payload TaskPayload) (*Task, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error
	CompleteTask(ctx context.Context, id uuid.UUID, result TaskResult) error
	FailTask(ctx context.Context, id uuid.UUID, errMsg string) error
}

type Service struct {
	Repository TaskRepository
	wg         sync.WaitGroup
}

func NewService(taskRepository TaskRepository) *Service {
	return &Service{
		Repository: taskRepository,
	}
}

func (s *Service) CreateTask(ctx context.Context, payload TaskPayload) (*Task, error) {
	task := &Task{
		ID:        uuid.New(),
		Status:    StatusInProgress,
		Payload:   payload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.Repository.CreateTask(ctx, task); err != nil {
		return &Task{}, fmt.Errorf("save task in repository: %w", err)
	}

	logger := ctxlog.FromContext(ctx)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processTask(task.ID, logger) // имитация обработки в фоне
	}()

	return task, nil
}

func (s *Service) processTask(taskID uuid.UUID, logger *slog.Logger) {
	ctx := context.Background()

	time.Sleep(time.Second * 12)

	result := TaskResult{OutputPath: "/tmp/fake_output.mp4"}
	if err := s.CompleteTask(ctx, taskID, result); err != nil {
		logger.Error("complete task failed", "task_id", taskID, "error", err)
	}
}

func (s *Service) GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error) {
	task, err := s.Repository.GetTaskByID(ctx, id)
	if err != nil {
		return Task{}, fmt.Errorf("get task from repository: %w", err)
	}

	return task, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error {
	return s.mutateTask(ctx, id, func(t *Task) {
		t.Status = status
	})
}

func (s *Service) CompleteTask(ctx context.Context, id uuid.UUID, result TaskResult) error {
	return s.mutateTask(ctx, id, func(t *Task) {
		t.Status = StatusReady
		t.Result = &result
		t.ErrorMsg = ""
	})
}

func (s *Service) FailTask(ctx context.Context, id uuid.UUID, errMsg string) error {
	return s.mutateTask(ctx, id, func(t *Task) {
		t.Status = StatusError
		t.Result = nil
		t.ErrorMsg = errMsg
	})
}

func (s *Service) mutateTask(ctx context.Context, id uuid.UUID, mutate func(*Task)) error {
	task, err := s.Repository.GetTaskByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get task from repository: %w", err)
	}
	mutate(&task)
	task.UpdatedAt = time.Now()
	if err := s.Repository.UpdateTask(ctx, &task); err != nil {
		return fmt.Errorf("update task in repository: %w", err)
	}
	return nil
}

func (s *Service) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
