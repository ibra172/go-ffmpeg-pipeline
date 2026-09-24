package task

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/ctxlog"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/httpresp"
)

type Handler struct {
	service       TaskService
	dataDir       string // путь для сохранения файла
	maxUploadSize int64  // максимальный размер входного файла в байтах
}

func NewHandler(service TaskService, dataDir string, maxUploadSizeMB int64) *Handler {
	return &Handler{
		service:       service,
		dataDir:       dataDir,
		maxUploadSize: maxUploadSizeMB << 20,
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

// TaskResultResponse is returned by the result endpoint.
// swagger:model TaskResultResponse
type TaskResultResponse struct {
	Status     string `json:"status"`
	OutputPath string `json:"output_path,omitempty"`
	Error      string `json:"error,omitempty"`
}

const (
	inMemoryFormBufferSize int64 = 10 << 20 // размер буфера парсинга multipart-формы в памяти
)

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
// @Failure 405 {object} httpresp.ErrorResponse "Invalid method"
// @Failure 500 {object} httpresp.ErrorResponse "Internal server error"
// @Router /task [post]
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := ctxlog.FromContext(ctx)

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		httpresp.RespondError(ctx, w, apperr.ErrMethodNotAllowed, "method not supported by the target URI")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)

	if err := r.ParseMultipartForm(inMemoryFormBufferSize); err != nil {
		httpresp.RespondError(
			ctx, w,
			fmt.Errorf("parse multipart form: %w: %w", err, apperr.ErrInvalidArgument),
			"the file is too large, or the request body is malformed",
		)
		return
	}
	defer r.MultipartForm.RemoveAll()

	req := CreateTaskRequest{
		Operation:    r.FormValue("operation"),
		TargetFormat: r.FormValue("target_format"),
		Resolution:   r.FormValue("resolution"),
	}

	if err := validateCreateTaskRequest(req); err != nil {
		httpresp.RespondError(ctx, w, err, "invalid task parameters")
		return
	}

	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		httpresp.RespondError(
			ctx, w,
			fmt.Errorf("read 'video' field: %w: %w", err, apperr.ErrInvalidArgument),
			"video file is required",
		)
		return
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, err := file.Read(sniff)
	if err != nil && err != io.EOF {
		httpresp.RespondError(ctx, w, fmt.Errorf("read uploaded file: %w", err), "internal error")
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		httpresp.RespondError(ctx, w, fmt.Errorf("reset file offset: %w", err), "internal error")
		return
	}

	contentType := http.DetectContentType(sniff[:n])
	if !strings.HasPrefix(contentType, "video/") {
		httpresp.RespondError(
			ctx, w,
			fmt.Errorf("uploaded content type %q is not a video: %w", contentType, apperr.ErrInvalidArgument),
			"uploaded file does not appear to be a video",
		)
		return
	}

	uploadsDir := filepath.Join(h.dataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		httpresp.RespondError(ctx, w, fmt.Errorf("create uploads dir: %w", err), "internal error")
		return
	}

	ext := filepath.Ext(fileHeader.Filename)
	dstPath := filepath.Join(uploadsDir, uuid.New().String()+ext)

	dst, err := os.Create(dstPath)
	if err != nil {
		httpresp.RespondError(ctx, w, fmt.Errorf("create destination file: %w", err), "internal error")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		httpresp.RespondError(ctx, w, fmt.Errorf("write file to disk: %w", err), "error saving uploaded file")
		return
	}

	logger.Info("file uploaded",
		"filename", fileHeader.Filename,
		"size", fileHeader.Size,
		"content_type", contentType,
		"saved_as", dstPath,
	)

	payload := Payload{
		Operation:    req.Operation,
		TargetFormat: req.TargetFormat,
		Resolution:   req.Resolution,
		InputPath:    dstPath,
	}

	createdTask, err := h.service.CreateTask(ctx, payload)
	if err != nil {
		httpresp.RespondError(ctx, w, err, "failed to create task")
		return
	}

	httpresp.RespondJSON(w, http.StatusCreated, CreateTaskResponse{
		TaskID: createdTask.ID.String(),
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

	task, err := h.service.GetTaskByID(r.Context(), id)
	if err != nil {
		httpresp.RespondError(ctx, w, err, "failed to get task")
		return
	}

	switch task.Status {
	case StatusInProgress:
		httpresp.RespondJSON(w, http.StatusAccepted, TaskStatusResponse{Status: "in_progress"})
	case StatusReady:
		httpresp.RespondJSON(w, http.StatusOK, TaskStatusResponse{Status: "ready"})
	case StatusError:
		httpresp.RespondJSON(w, http.StatusOK, TaskStatusResponse{Status: "error"})
	}

}

// GetResult returns the result payload for a completed task.
// @Summary Get task result
// @Description Returns the processing result for a task when it is finished. For failed tasks, the API returns the error message instead of the output artifact.
// @Tags task
// @Produce json
// @Param id path string true "Task ID in UUID format" example(123e4567-e89b-12d3-a456-426614174000)
// @Security BearerAuth
// @Success 200 {object} TaskResultResponse "Task finished (see status field for success/error)"
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

	task, err := h.service.GetTaskByID(r.Context(), id)
	if err != nil {
		httpresp.RespondError(ctx, w, err, "failed to get task")
		return
	}

	switch task.Status {
	case StatusError:
		httpresp.RespondJSON(w, http.StatusOK, TaskResultResponse{
			Status: "error",
			Error:  task.ErrorMsg,
		})
	case StatusReady:
		httpresp.RespondJSON(w, http.StatusOK, TaskResultResponse{
			Status:     "ready",
			OutputPath: task.Result.OutputPath,
		})
	default: // StatusInProgress
		httpresp.RespondJSON(w, http.StatusAccepted, TaskStatusResponse{
			Status: "in_progress",
		})
	}
}
