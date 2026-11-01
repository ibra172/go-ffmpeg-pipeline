package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
)

func TestGetTaskForUser(t *testing.T) {
	ctx := context.Background()
	repo := NewRamRepository()
	service := NewService(repo, nil, "")

	ownerUser, otherUser := uuid.New(), uuid.New()
	usersTask := Task{ID: uuid.New(), UserID: ownerUser}

	if err := repo.CreateTask(ctx, &usersTask); err != nil {
		t.Fatalf("create fixture task: %v", err)
	}

	t.Run("ExistingTaskOwnerUser", func(t *testing.T) {
		task, err := service.GetTaskForUser(ctx, usersTask.ID, ownerUser)
		if err != nil || task.ID != usersTask.ID {
			t.Fatalf(
				"expected task ID %s and no error, got ID %s, error %v",
				usersTask.ID, task.ID, err,
			)
		}
	})

	t.Run("ExistingTaskOtherUser", func(t *testing.T) {
		_, err := service.GetTaskForUser(ctx, usersTask.ID, otherUser)
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("NonExistentTask", func(t *testing.T) {
		_, err := service.GetTaskForUser(ctx, uuid.New(), ownerUser)
		if !errors.Is(err, apperr.ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

// Каждый подтест проверяет отдельный сценарий и получает своё хранилище.
// t.Run группирует сценарии; сами проверки выполняются через t.Fatal/t.Errorf.
func TestTaskStateTransitions(t *testing.T) {
	t.Run("ready task rejects failure", func(t *testing.T) {
		ctx := context.Background()
		service, repo, taskID := newTaskStateFixture(t)
		const outputPath = "/data/results/input_preview.jpg"

		// Сначала успешно завершаем задачу обычным публичным методом service.
		if err := service.CompleteTask(ctx, taskID, Result{OutputPath: outputPath}); err != nil {
			t.Fatalf("complete task: %v", err)
		}
		completed := mustGetTask(t, repo, taskID)

		// Повторная попытка изменить конечное состояние должна быть отклонена.
		err := service.FailTask(ctx, taskID, "late ffmpeg error")
		if !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("FailTask() error = %v, want ErrConflict", err)
		}

		// Проверяем сохранённые данные: правильной ошибки самой по себе мало.
		got := mustGetTask(t, repo, taskID)
		if got.Status != StatusReady {
			t.Errorf("status = %q, want %q", got.Status, StatusReady)
		}
		if got.Result == nil {
			t.Fatal("result is nil, want successful result")
		}
		if got.Result.OutputPath != outputPath {
			t.Errorf("output path = %q, want %q", got.Result.OutputPath, outputPath)
		}
		if got.ErrorMsg != "" {
			t.Errorf("error message = %q, want empty", got.ErrorMsg)
		}
		if !got.UpdatedAt.Equal(completed.UpdatedAt) {
			t.Errorf("updated_at changed after rejected update: got %v, want %v", got.UpdatedAt, completed.UpdatedAt)
		}
	})

	t.Run("failed task rejects success", func(t *testing.T) {
		ctx := context.Background()
		service, repo, taskID := newTaskStateFixture(t)
		const errorMessage = "ffmpeg could not decode input"

		if err := service.FailTask(ctx, taskID, errorMessage); err != nil {
			t.Fatalf("fail task: %v", err)
		}
		failed := mustGetTask(t, repo, taskID)

		err := service.CompleteTask(ctx, taskID, Result{OutputPath: "/data/results/late.jpg"})
		if !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("CompleteTask() error = %v, want ErrConflict", err)
		}

		got := mustGetTask(t, repo, taskID)
		if got.Status != StatusError {
			t.Errorf("status = %q, want %q", got.Status, StatusError)
		}
		if got.Result != nil {
			t.Errorf("result = %+v, want nil", got.Result)
		}
		if got.ErrorMsg != errorMessage {
			t.Errorf("error message = %q, want %q", got.ErrorMsg, errorMessage)
		}
		if !got.UpdatedAt.Equal(failed.UpdatedAt) {
			t.Errorf("updated_at changed after rejected update: got %v, want %v", got.UpdatedAt, failed.UpdatedAt)
		}
	})

	t.Run("empty result leaves task in progress", func(t *testing.T) {
		ctx := context.Background()
		service, repo, taskID := newTaskStateFixture(t)
		initial := mustGetTask(t, repo, taskID)

		err := service.CompleteTask(ctx, taskID, Result{})
		if !errors.Is(err, apperr.ErrInvalidArgument) {
			t.Fatalf("CompleteTask() error = %v, want ErrInvalidArgument", err)
		}

		got := mustGetTask(t, repo, taskID)
		if got.Status != StatusInProgress {
			t.Errorf("status = %q, want %q", got.Status, StatusInProgress)
		}
		if got.Result != nil {
			t.Errorf("result = %+v, want nil", got.Result)
		}
		if got.ErrorMsg != "" {
			t.Errorf("error message = %q, want empty", got.ErrorMsg)
		}
		if !got.UpdatedAt.Equal(initial.UpdatedAt) {
			t.Errorf("updated_at changed after rejected result: got %v, want %v", got.UpdatedAt, initial.UpdatedAt)
		}
	})
}

// Фикстура — начальные данные теста. Каждый вызов создаёт независимую задачу.
func newTaskStateFixture(t *testing.T) (*Service, *RamRepository, uuid.UUID) {
	// Ошибка внутри helper будет указывать на строку его вызова в тесте.
	t.Helper()

	repo := NewRamRepository()
	now := time.Now()
	task := Task{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Status:    StatusInProgress,
		Payload:   Payload{Operation: "thumbnail", InputPath: "/data/uploads/input.mp4"},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateTask(context.Background(), &task); err != nil {
		t.Fatalf("create fixture task: %v", err)
	}

	// Проверяем чтение и завершение; отправитель сообщений здесь не вызывается.
	return NewService(repo, nil, ""), repo, task.ID
}

func mustGetTask(t *testing.T, repo TaskRepository, taskID uuid.UUID) Task {
	t.Helper()

	got, err := repo.GetTaskByID(context.Background(), taskID)
	if err != nil {
		// Без прочитанной задачи продолжать проверки её полей бессмысленно.
		t.Fatalf("read stored task %s: %v", taskID, err)
	}
	return got
}
