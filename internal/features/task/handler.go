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

// CreateTask creates a new media-processing task.
// @Summary Create processing task
// @Description Creates a new media-processing job. The task is queued immediately and can be checked by its UUID through the status and result endpoints.
// @Tags task
// @Accept json
// @Produce json
// @Param request body TaskPayload true "Task creation payload" example({"operation":"transcode","target_format":"mp4","resolution":"720p"})
// @Success 201 {object} CreateTaskResponse "Task created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /task [post]
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var payload TaskPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil && !errors.Is(err, io.EOF) {
		httpresp.RespondError(
			ctx,
			w,
			fmt.Errorf("%w: %s", apperr.ErrInvalidArgument, err),
			"invalid request body",
		)

		return
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
// @Success 200 {object} TaskStatusResponse "Current task status"
// @Success 202 {object} TaskStatusResponse "Task is still processing"
// @Failure 400 {object} ErrorResponse "Invalid UUID format"
// @Failure 404 {object} ErrorResponse "Task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
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
		if errors.Is(err, apperr.ErrNotFound) {
			httpresp.RespondError(ctx, w, err, "task not found")
			return
		}
		httpresp.RespondError(ctx, w, err, "failed to get task")
		return
	}

	httpresp.RespondJSON(w, http.StatusAccepted, map[string]string{
		"status": externalStatus(task.Status),
	})
}

// GetResult returns the result payload for a completed task.
// @Summary Get task result
// @Description Returns the processing result for a task when it is finished. For failed tasks, the API returns the error message instead of the output artifact.
// @Tags task
// @Produce json
// @Param id path string true "Task ID in UUID format" example(123e4567-e89b-12d3-a456-426614174000)
// @Success 200 {object} TaskResultResponse "Task finished successfully"
// @Failure 400 {object} ErrorResponse "Invalid UUID format"
// @Failure 404 {object} ErrorResponse "Task not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
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
		if errors.Is(err, apperr.ErrNotFound) {
			httpresp.RespondError(ctx, w, err, "task not found")
			return
		}
		httpresp.RespondError(ctx, w, err, "failed to get task")

		return
	}

	if task.Status == StatusError {
		httpresp.RespondJSON(w, http.StatusOK, map[string]string{
			"status": "error",
			"error":  task.ErrorMsg,
		})

		return
	}

	httpresp.RespondJSON(w, http.StatusOK, TaskResultResponse{
		Result: *task.Result,
	})
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
