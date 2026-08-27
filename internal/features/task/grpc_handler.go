package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/grpc/taskpb"
)

type GRPCHandler struct {
	taskpb.UnimplementedTaskServiceServer
	service TaskService
}

func NewGRPCHandler(service TaskService) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

func (h *GRPCHandler) CommitTaskResult(
	ctx context.Context,
	taskMsgRequest *taskpb.TaskMessageRequest,
) (*taskpb.TaskMessageResponse, error) {
	id, err := uuid.Parse(taskMsgRequest.GetTaskId())
	if err != nil {
		return &taskpb.TaskMessageResponse{Success: false},
			fmt.Errorf("parse task_id into a UUID: %w", err)
	}

	result := taskMsgRequest.GetResult()

	switch taskMsgRequest.GetStatus() {
	case taskpb.Status_STATUS_READY:
		if err := h.service.CompleteTask(ctx, id, Result{OutputPath: result.GetOutputPath()}); err != nil {
			return &taskpb.TaskMessageResponse{Success: false}, fmt.Errorf("complete task: %w", err)
		}
	case taskpb.Status_STATUS_ERROR:
		if err := h.service.FailTask(ctx, id, result.GetErrorMsg()); err != nil {
			return &taskpb.TaskMessageResponse{Success: false}, fmt.Errorf("fail task: %w", err)
		}
	default:
		return &taskpb.TaskMessageResponse{Success: false},
			fmt.Errorf("unexpected status: %v", taskMsgRequest.GetStatus())
	}

	return &taskpb.TaskMessageResponse{Success: true}, nil
}
