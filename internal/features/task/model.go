package task

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus — статус задачи. StatusCreated используется только внутри
// домена (задача создана, но воркер её ещё не забрал из очереди в 3-й части).
// Наружу в /status этот статус нужно маппить в "in_progress" — тесты
// ожидают только "in_progress" | "ready".
type TaskStatus string

const (
	StatusInProgress TaskStatus = "in_progress"
	StatusReady      TaskStatus = "ready"
	StatusError      TaskStatus = "error"
)

// TaskPayload represents the request body used to create a media-processing task.
// swagger:model TaskPayload
type TaskPayload struct {
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
	// Result contains the processing output.
	Result TaskResult `json:"result"`
}

// ErrorResponse describes an API error payload.
// swagger:model ErrorResponse
type ErrorResponse struct {
	// Error holds a public error message.
	Error string `json:"error"`
}

// TaskResult represents successful processing output for a task.
// swagger:model TaskResult
type TaskResult struct {
	// OutputPath is the resulting artifact path.
	OutputPath string `json:"output_path"`
}

// Task — доменная модель задачи.
type Task struct {
	ID      uuid.UUID   `json:"id"`
	Status  TaskStatus  `json:"status"`
	Payload TaskPayload `json:"payload"`

	Result   *TaskResult `json:"result,omitempty"`
	ErrorMsg string      `json:"error_msg,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
