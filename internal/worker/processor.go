package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
)

type Processor struct {
	logger        *slog.Logger
	dataDir       string
	watermarkPath string
	ffmpegTimeout time.Duration
	taskClient    TaskClient
}

type TaskClient interface {
	CommitSuccess(ctx context.Context, taskID uuid.UUID, result task.Result) error
	CommitFailure(ctx context.Context, taskID uuid.UUID, errMsg string) error
	GetStatus(ctx context.Context, taskID uuid.UUID) (task.Status, error)
}

func NewProcessor(logger *slog.Logger, dataDir string, watermarkPath string, ffmpegTimeout time.Duration, taskClient TaskClient) *Processor {
	return &Processor{
		logger:        logger,
		dataDir:       dataDir,
		watermarkPath: watermarkPath,
		ffmpegTimeout: ffmpegTimeout,
		taskClient:    taskClient,
	}
}

func (p *Processor) Process(ctx context.Context, msg task.Message) error {
	p.logger.Info("processing task",
		"task_id", msg.TaskID,
		"operation", msg.Operation,
		"input_path", msg.InputPath,
	)

	taskStatus, err := p.taskClient.GetStatus(ctx, msg.TaskID)
	if err != nil {
		return fmt.Errorf("get task status before running ffmpeg command: %w", err)
	}
	if taskStatus == task.StatusReady || taskStatus == task.StatusError {
		p.logger.Warn("task already processed", "task_id", msg.TaskID, "status", taskStatus)
		return nil
	}

	ffmpegCtx, cancel := context.WithTimeout(ctx, p.ffmpegTimeout)
	defer cancel()

	outputPath, args, err := buildFFmpegCommand(msg, p.dataDir, p.watermarkPath)
	if err != nil {
		if commitErr := p.taskClient.CommitFailure(ctx, msg.TaskID, err.Error()); commitErr != nil {
			return fmt.Errorf("build ffmpeg command: %w; commit failure also failed: %w", err, commitErr)
		}

		p.logger.Error("task failed during building ffmpeg command", "task_id", msg.TaskID, "error", err)
		return nil
	}

	if err := runFFmpeg(ffmpegCtx, args); err != nil {
		if commitErr := p.taskClient.CommitFailure(ctx, msg.TaskID, err.Error()); commitErr != nil {
			return fmt.Errorf("run ffmpeg: %w; commit failure also failed: %w", err, commitErr)
		}

		if ffmpegCtx.Err() == context.DeadlineExceeded {
			p.logger.Error("task failed: ffmpeg exceeded timeout", "task_id", msg.TaskID, "timeout", p.ffmpegTimeout)
		} else {
			p.logger.Error("task failed during ffmpeg processing", "task_id", msg.TaskID, "error", err)
		}

		return nil
	}

	result := task.Result{OutputPath: outputPath}
	if err := p.taskClient.CommitSuccess(ctx, msg.TaskID, result); err != nil {
		return fmt.Errorf("commit success: %w", err)
	}

	return nil
}
