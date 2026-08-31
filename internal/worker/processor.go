package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
)

type Processor struct {
	logger        *slog.Logger
	dataDir       string
	committer     ResultCommitter
	watermarkPath string
}

type ResultCommitter interface {
	CommitSuccess(ctx context.Context, taskID uuid.UUID, result task.Result) error
	CommitFailure(ctx context.Context, taskID uuid.UUID, errMsg string) error
}

func NewProcessor(logger *slog.Logger, dataDir string, watermarkPath string, committer ResultCommitter) *Processor {
	return &Processor{
		logger:        logger,
		dataDir:       dataDir,
		watermarkPath: watermarkPath,
		committer:     committer,
	}
}

func (p *Processor) Process(ctx context.Context, msg task.Message) error {
	p.logger.Info("processing task",
		"task_id", msg.TaskID,
		"operation", msg.Operation,
		"input_path", msg.InputPath,
	)

	outputPath, args, err := buildFFmpegCommand(msg, p.dataDir, p.watermarkPath)
	if err != nil {
		if commitErr := p.committer.CommitFailure(ctx, msg.TaskID, err.Error()); commitErr != nil {
			return fmt.Errorf("build ffmpeg command: %w; commit failure also failed: %w", err, commitErr)
		}

		p.logger.Error("task failed during building ffmpeg command", "task_id", msg.TaskID, "error", err)
		return nil
	}

	if err := runFFmpeg(ctx, args); err != nil {
		if commitErr := p.committer.CommitFailure(ctx, msg.TaskID, err.Error()); commitErr != nil {
			return fmt.Errorf("run ffmpeg: %w; commit failure also failed: %w", err, commitErr)
		}

		p.logger.Error("task failed during ffmpeg processing", "task_id", msg.TaskID, "error", err)
		return nil
	}

	result := task.Result{OutputPath: outputPath}
	if err := p.committer.CommitSuccess(ctx, msg.TaskID, result); err != nil {
		return fmt.Errorf("commit success: %w", err)
	}

	return nil
}
