package task

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TaskService interface {
	CreateTask(ctx context.Context, payload TaskPayload) (*Task, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error
	CompleteTask(ctx context.Context, id uuid.UUID, result TaskResult) error
	FailTask(ctx context.Context, id uuid.UUID, errMsg string) error
}

type Service struct {
	repository TaskRepository
	sender     Sender
}

func NewService(taskRepository TaskRepository, sender Sender) *Service {
	return &Service{
		repository: taskRepository,
		sender:     sender,
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

	if err := s.repository.CreateTask(ctx, task); err != nil {
		return &Task{}, fmt.Errorf("save task in repository: %w", err)
	}

	msg := Message{
		TaskID:       task.ID,
		Operation:    task.Payload.Operation,
		TargetFormat: task.Payload.TargetFormat,
		Resolution:   task.Payload.Resolution,
		InputPath:    fmt.Sprintf("/data/uploads/{%s}.mp4", task.ID),
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
