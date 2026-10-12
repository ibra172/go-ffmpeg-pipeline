package grpc_client

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/grpc/taskpb"
)

const timeout = 5 * time.Second

type TaskClient struct {
	client taskpb.TaskServiceClient
}

func NewTaskClient(conn grpc.ClientConnInterface) *TaskClient {
	return &TaskClient{
		client: taskpb.NewTaskServiceClient(conn),
	}
}

func (c *TaskClient) CommitSuccess(ctx context.Context, taskID uuid.UUID, result task.Result) error {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req := &taskpb.TaskMessageRequest{
		TaskId: taskID.String(),
		Status: taskpb.Status_STATUS_READY,
		Result: &taskpb.Result{OutputPath: result.OutputPath},
	}
	_, err := c.client.CommitTaskResult(callCtx, req)
	if err != nil {
		if callCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gRPC timeout: %w", err)
		}
		return fmt.Errorf("commit success result: %w", err)
	}

	return nil
}

func (c *TaskClient) CommitFailure(ctx context.Context, taskID uuid.UUID, errMsg string) error {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req := &taskpb.TaskMessageRequest{
		TaskId: taskID.String(),
		Status: taskpb.Status_STATUS_ERROR,
		Result: &taskpb.Result{ErrorMsg: errMsg},
	}
	_, err := c.client.CommitTaskResult(callCtx, req)
	if err != nil {
		if callCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gRPC timeout: %w", err)
		}
		return fmt.Errorf("commit failure result: %w", err)
	}

	return nil
}

func (c *TaskClient) GetStatus(ctx context.Context, taskID uuid.UUID) (task.Status, error) {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req := &taskpb.GetTaskStatusRequest{
		TaskId: taskID.String(),
	}

	resp, err := c.client.GetTaskStatus(callCtx, req)
	if err != nil {
		if callCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("gRPC timeout: %w", err)
		}
		return "", fmt.Errorf("get task status: %w", err)
	}

	switch resp.Status {
	case taskpb.Status_STATUS_IN_PROGRESS:
		return task.StatusInProgress, nil
	case taskpb.Status_STATUS_READY:
		return task.StatusReady, nil
	case taskpb.Status_STATUS_ERROR:
		return task.StatusError, nil
	default:
		return "", fmt.Errorf("unknown status: %+v", resp.Status)
	}
}
