package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/apperr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewPostgresRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *PostgresRepository {
	return &PostgresRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *PostgresRepository) CreateTask(ctx context.Context, task *Task) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query := `
	INSERT INTO ffmpeg_app.tasks (
    	id, user_id, status,
    	operation, target_format, resolution, input_path,
    	created_at, updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.pool.Exec(
		queryCtx,
		query,
		task.ID, task.UserID, task.Status,
		task.Payload.Operation, task.Payload.TargetFormat,
		task.Payload.Resolution, task.Payload.InputPath,
		task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("execute create task query: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query := `
	SELECT id, user_id, status,
       operation, target_format, resolution, input_path,
       output_path, error_msg, created_at, updated_at
	FROM ffmpeg_app.tasks
	WHERE id = $1
	`

	var task Task
	var outputPath string

	err := r.pool.QueryRow(queryCtx, query, id).Scan(
		&task.ID, &task.UserID, &task.Status,
		&task.Payload.Operation, &task.Payload.TargetFormat,
		&task.Payload.Resolution, &task.Payload.InputPath,
		&outputPath, &task.ErrorMsg,
		&task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Task{}, fmt.Errorf("no task with ID=`%v`: %w", id, apperr.ErrNotFound)
		}

		return Task{}, fmt.Errorf("scan row: %w", err)
	}

	if task.Status == StatusReady {
		task.Result = &Result{OutputPath: outputPath}
	}

	return task, nil
}

func (r *PostgresRepository) UpdateTask(ctx context.Context, task *Task) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query := `
	UPDATE ffmpeg_app.tasks
	SET status = $2,
    	output_path = $3,
    	error_msg = $4,
    	updated_at = $5
	WHERE id = $1 AND status = 'in_progress'
	`

	var outputPath string
	if task.Result != nil {
		outputPath = task.Result.OutputPath
	}

	tag, err := r.pool.Exec(
		queryCtx,
		query,
		task.ID,
		task.Status,
		outputPath,
		task.ErrorMsg,
		task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("execute update task query: %w", err)
	}

	if tag.RowsAffected() != 1 {
		_, err := r.GetTaskByID(queryCtx, task.ID)
		if err != nil {
			return err
		}

		return fmt.Errorf("task cannot be updated: %w", apperr.ErrConflict)
	}

	return nil
}
