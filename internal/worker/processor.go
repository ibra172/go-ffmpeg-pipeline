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
	logger   *slog.Logger
	dataDir  string
	committer ResultCommitter
}

type ResultCommitter interface {
	CommitSuccess(ctx context.Context, taskID uuid.UUID, result task.Result) error
	CommitFailure(ctx context.Context, taskID uuid.UUID, errMsg string) error
}

func NewProcessor(logger *slog.Logger, dataDir string, commiter ResultCommitter) *Processor {
	return &Processor{
		logger:   logger,
		dataDir:  dataDir,
		committer: commiter,
	}
}

func (p *Processor) Process(ctx context.Context, msg task.Message) error {
	p.logger.Info("processing task",
		"task_id", msg.TaskID,
		"operation", msg.Operation,
		"input_path", msg.InputPath,
	)

	time.Sleep(time.Second * 2) // TODO: реальный ffmpeg

	outputPath := "fake output path"

	result := task.Result{OutputPath: outputPath}
	if err := p.committer.CommitSuccess(ctx, msg.TaskID, result); err != nil {
		return fmt.Errorf("commit success: %w", err)
	}

	return nil
}
