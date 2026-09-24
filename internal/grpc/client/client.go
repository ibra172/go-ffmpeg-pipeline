package grpc_client

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/grpc/taskpb"
)

type resultCommitter struct {
	client taskpb.TaskServiceClient
}

func NewResultCommiter(conn grpc.ClientConnInterface) *resultCommitter {
	return &resultCommitter{
		client: taskpb.NewTaskServiceClient(conn),
	}
}

func (c *resultCommitter) CommitSuccess(ctx context.Context, taskID uuid.UUID, result task.Result) error {
	req := &taskpb.TaskMessageRequest{
		TaskId: taskID.String(),
		Status: taskpb.Status_STATUS_READY,
		Result: &taskpb.Result{OutputPath: result.OutputPath},
	}
	_, err := c.client.CommitTaskResult(ctx, req)
	if err != nil {
		return fmt.Errorf("commit success result: %w", err)
	}

	return nil
}

func (c *resultCommitter) CommitFailure(ctx context.Context, taskID uuid.UUID, errMsg string) error {
	req := &taskpb.TaskMessageRequest{
		TaskId: taskID.String(),
		Status: taskpb.Status_STATUS_ERROR,
		Result: &taskpb.Result{ErrorMsg: errMsg},
	}
	_, err := c.client.CommitTaskResult(ctx, req)
	if err != nil {
		return fmt.Errorf("commit failure result: %w", err)
	}

	return nil
}
