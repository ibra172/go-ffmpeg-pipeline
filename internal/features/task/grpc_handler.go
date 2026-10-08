package task

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/grpc/taskpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, status.Errorf(codes.InvalidArgument, "parse task_id into a UUID: %v", err)
	}

	result := taskMsgRequest.GetResult()

	switch taskMsgRequest.GetStatus() {
	case taskpb.Status_STATUS_READY:
		if err := h.service.CompleteTask(ctx, id, Result{OutputPath: result.GetOutputPath()}); err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return nil, status.Errorf(codes.NotFound, "task %s not found", id)
			}
			return nil, status.Errorf(codes.Internal, "complete task: %v", err)
		}
	case taskpb.Status_STATUS_ERROR:
		if err := h.service.FailTask(ctx, id, result.GetErrorMsg()); err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return nil, status.Errorf(codes.NotFound, "task %s not found", id)
			}
			return nil, status.Errorf(codes.Internal, "fail task: %v", err)
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unexpected status: %v", taskMsgRequest.GetStatus())
	}

	return &taskpb.TaskMessageResponse{}, nil
}

func (h *GRPCHandler) GetTaskStatus(
	ctx context.Context,
	req *taskpb.GetTaskStatusRequest,
) (*taskpb.GetTaskStatusResponse, error) {
	taskID, err := uuid.Parse(req.GetTaskId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parse task_id into a UUID: %v", err)
	}

	task, err := h.service.GetTaskByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "task %s not found", taskID)
		}
		return nil, status.Errorf(codes.Internal, "get task from service: %v", err)
	}

	var statusResponse taskpb.GetTaskStatusResponse

	switch task.Status {
	case StatusInProgress:
		statusResponse.Status = taskpb.Status_STATUS_IN_PROGRESS
	case StatusReady:
		statusResponse.Status = taskpb.Status_STATUS_READY
	case StatusError:
		statusResponse.Status = taskpb.Status_STATUS_ERROR
	default:
		return nil, status.Errorf(codes.Internal, "unexpected status: %v", task.Status)
	}

	return &statusResponse, nil
}
