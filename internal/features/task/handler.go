package task

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/httpresp"
)

type Handler struct {
	Service TaskService
}

func NewHandler(service TaskService) *Handler {
	return &Handler{
		Service: service,
	}
}

// CreateTaskRequest represents the request body used to create a media-processing task.
// swagger:model CreateTaskRequest
type CreateTaskRequest struct {
	// Operation defines the type of job to run. Examples: transcode, thumbnail, watermark.
	Operation string `json:"operation"`
	// TargetFormat is used by transcode tasks and may be set to mp4, mov, etc.
	TargetFormat string `json:"target_format,omitempty"`
	// Resolution is used by transcode tasks and matches dimension presets such as 720p.
	Resolution string `json:"resolution,omitempty"`
}

// CreateTaskResponse is returned when a new task has been successfully created.
// swagger:model CreateTaskResponse
type CreateTaskResponse struct {
	// TaskID is the generated UUID of the created task.
	TaskID string `json:"task_id"`
}

// TaskStatusResponse is returned by the status endpoint.
// swagger:model TaskStatusResponse
type TaskStatusResponse struct {
	// Status can be in_progress, ready, or error.
	Status string `json:"status"`
}

// TaskErrorResponse is returned when a task fails and the API still responds with HTTP 200.
// swagger:model TaskErrorResponse
type TaskErrorResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// TaskResultResponse is returned by the result endpoint.
// swagger:model TaskResultResponse
type TaskResultResponse struct {
	OutputPath string `json:"output_path"`
}

// CreateTask creates a new media-processing task.
// @Summary Create processing task
// @Description Creates a new media-processing job. The task is queued immediately and can be checked by its UUID through the status and result endpoints.
// @Tags task
// @Accept json
// @Produce json
// @Param request body CreateTaskRequest true "Task creation payload" example({"operation":"transcode","target_format":"mp4","resolution":"720p"})
// @Security BearerAuth
// @Success 201 {object} CreateTaskResponse "Task created successfully"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid request body"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /task [post]
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreateTaskRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil && !errors.Is(err, io.EOF) {
		httpresp.RespondError(
			ctx,
			w,
			fmt.Errorf("%w: %s", apperr.ErrInvalidArgument, err),
			"invalid request body",
		)
		return
	}

	payload := TaskPayload{
		Operation:    req.Operation,
		TargetFormat: req.TargetFormat,
		Resolution:   req.Resolution,
	}

	task, err := h.Service.CreateTask(ctx, payload)
	if err != nil {
		httpresp.RespondError(ctx, w, err, "failed to create task")
		return
	}

	httpresp.RespondJSON(w, http.StatusCreated, CreateTaskResponse{
		TaskID: task.ID.String(),
	})
}

// GetStatus returns the current status of a task by its ID.
// @Summary Get task status
// @Description Returns the current processing state of a task. The public status values are in_progress, ready, and error.
// @Tags task
// @Produce json
// @Param id path string true "Task ID in UUID format" example(123e4567-e89b-12d3-a456-426614174000)
// @Security BearerAuth
// @Success 200 {object} TaskStatusResponse "Current task status"
// @Success 202 {object} TaskStatusResponse "Task is still processing"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid UUID format"
// @Failure 404 {object} httpresp.ErrorResponse "Task not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /status/{id} [get]
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresp.RespondError(
			ctx,
			w,
			fmt.Errorf("%w: %s", apperr.ErrInvalidArgument, err),
			"invalid task id",
		)
		return
	}

	task, err := h.Service.GetTaskByID(r.Context(), id)
	if err != nil {
		httpresp.RespondError(ctx, w, err, "failed to get task")
		return
	}

	httpresp.RespondJSON(w, http.StatusAccepted, TaskStatusResponse{
		Status: externalStatus(task.Status),
	})
}

// GetResult returns the result payload for a completed task.
// @Summary Get task result
// @Description Returns the processing result for a task when it is finished. For failed tasks, the API returns the error message instead of the output artifact.
// @Tags task
// @Produce json
// @Param id path string true "Task ID in UUID format" example(123e4567-e89b-12d3-a456-426614174000)
// @Security BearerAuth
// @Success 200 {object} TaskErrorResponse "Task finished with an error"
// @Success 200 {object} TaskResultResponse "Task finished successfully"
// @Success 202 {object} TaskStatusResponse "Task is still being processed"
// @Failure 400 {object} httpresp.ErrorResponse "Invalid UUID format"
// @Failure 404 {object} httpresp.ErrorResponse "Task not found"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /result/{id} [get]
func (h *Handler) GetResult(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresp.RespondError(
			ctx,
			w,
			fmt.Errorf("%w: %s", apperr.ErrInvalidArgument, err),
			"invalid task id",
		)
		return
	}

	task, err := h.Service.GetTaskByID(r.Context(), id)
	if err != nil {
		httpresp.RespondError(ctx, w, err, "failed to get task")
		return
	}

	switch task.Status {
	case StatusError:
		httpresp.RespondJSON(w, http.StatusOK, TaskErrorResponse{
			Status: "error",
			Error:  task.ErrorMsg,
		})
	case StatusReady:
		httpresp.RespondJSON(w, http.StatusOK, TaskResultResponse{
			OutputPath: task.Result.OutputPath,
		})
	default: // StatusInProgress
		httpresp.RespondJSON(w, http.StatusAccepted, TaskStatusResponse{
			Status: "in_progress",
		})
	}
}

func externalStatus(s TaskStatus) string {
	switch s {
	case StatusReady:
		return "ready"
	case StatusError:
		return "error"
	default:
		return "in_progress"
	}
}
