package worker

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
)

// Реализует TaskClient без настоящих gRPC-вызовов.
type taskClientStub struct {
	status task.Status

	getStatusCalls     int
	commitSuccessCalls int
	commitFailureCalls int
}

func (c *taskClientStub) GetStatus(
	context.Context,
	uuid.UUID,
) (task.Status, error) {
	c.getStatusCalls++
	return c.status, nil
}

func (c *taskClientStub) CommitSuccess(
	context.Context,
	uuid.UUID,
	task.Result,
) error {
	c.commitSuccessCalls++
	return nil
}

func (c *taskClientStub) CommitFailure(
	context.Context,
	uuid.UUID,
	string,
) error {
	c.commitFailureCalls++
	return nil
}

func TestProcessorSkipsTerminalTask(t *testing.T) {
	statuses := []task.Status{
		task.StatusReady,
		task.StatusError,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			// У каждого сценария отдельный клиент и счётчики.
			client := &taskClientStub{status: status}

			processor := NewProcessor(
				slog.Default(),
				t.TempDir(),
				"",
				time.Second,
				client,
			)

			msg := task.Message{
				TaskID:    uuid.New(),
				Operation: "invalid",
			}

			err := processor.Process(context.Background(), msg)
			if err != nil {
				t.Fatalf("Process() error = %v, want nil", err)
			}

			if client.getStatusCalls != 1 {
				t.Errorf(
					"GetStatus calls = %d, want 1",
					client.getStatusCalls,
				)
			}

			if client.commitSuccessCalls != 0 {
				t.Errorf(
					"CommitSuccess calls = %d, want 0",
					client.commitSuccessCalls,
				)
			}

			if client.commitFailureCalls != 0 {
				t.Errorf(
					"CommitFailure calls = %d, want 0",
					client.commitFailureCalls,
				)
			}
		})
	}
}
