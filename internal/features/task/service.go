package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
)

type TaskService interface {
	CreateTask(ctx context.Context, userID uuid.UUID, payload Payload) (*Task, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error)
	GetTaskForUser(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) (Task, error)
	CompleteTask(ctx context.Context, id uuid.UUID, result Result) error
	FailTask(ctx context.Context, id uuid.UUID, errMsg string) error
}

type Service struct {
	repository TaskRepository
	sender     Sender
	dataDir    string
}

func NewService(taskRepository TaskRepository, sender Sender, dataDir string) *Service {
	return &Service{
		repository: taskRepository,
		sender:     sender,
		dataDir:    dataDir,
	}
}

func (s *Service) CreateTask(ctx context.Context, userID uuid.UUID, payload Payload) (*Task, error) {
	task := &Task{
		ID:        uuid.New(),
		Status:    StatusInProgress,
		Payload:   payload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    userID,
	}

	if err := s.repository.CreateTask(ctx, task); err != nil {
		return &Task{}, fmt.Errorf("save task in repository: %w", err)
	}

	msg := Message{
		TaskID:       task.ID,
		Operation:    task.Payload.Operation,
		TargetFormat: task.Payload.TargetFormat,
		Resolution:   task.Payload.Resolution,
		InputPath:    task.Payload.InputPath,
	}

	if err := s.sender.Send(ctx, msg); err != nil {
		_ = s.FailTask(ctx, task.ID, "failed to queue the task: "+err.Error())
		return &Task{}, fmt.Errorf("send task message: %w", err)
	}

	return task, nil
}

func (s *Service) GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error) {
	task, err := s.repository.GetTaskByID(ctx, id)
	if err != nil {
		return Task{}, fmt.Errorf("get task from repository: %w", err)
	}

	return task, nil
}

func (s *Service) GetTaskForUser(
	ctx context.Context,
	taskID uuid.UUID,
	userID uuid.UUID,
) (Task, error) {
	task, err := s.GetTaskByID(ctx, taskID)
	if err != nil {
		return Task{}, fmt.Errorf("get task by id: %w", err)
	}
	if task.UserID != userID {
		return Task{}, fmt.Errorf(
			"task '%s' does not belong to user '%s': %w",
			taskID,
			userID,
			apperr.ErrNotFound,
		)
	}

	return task, nil
}

func (s *Service) CompleteTask(ctx context.Context, id uuid.UUID, result Result) error {
	if result.OutputPath == "" {
		return fmt.Errorf("output path is required: %w", apperr.ErrInvalidArgument)
	}
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
	task, err := s.repository.GetTaskByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get task from repository: %w", err)
	}
	mutate(&task)
	task.UpdatedAt = time.Now()
	if err := s.repository.UpdateTask(ctx, &task); err != nil {
		return fmt.Errorf("update task in repository: %w", err)
	}
	return nil
}
