package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
)

type Processor struct {
	logger  *slog.Logger
	dataDir string
}

func NewProcessor(logger *slog.Logger, dataDir string) *Processor {
	return &Processor{logger: logger, dataDir: dataDir}
}

func (p *Processor) Process(ctx context.Context, msg task.Message) error {
	p.logger.Info("processing task",
		"task_id", msg.TaskID,
		"operation", msg.Operation,
		"input_path", msg.InputPath,
	)

	time.Sleep(time.Second * 2) // TODO: реальный ffmpeg появится позже

	return nil
}
