package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/config"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
	grpc_client "github.com/ibra172/go-ffmpeg-pipeline/internal/grpc/client"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/queue/rabbitmq"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/worker"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.MustNew()

	rabbitClient, err := rabbitmq.NewClient(cfg.AMQPURL)
	if err != nil {
		return fmt.Errorf("failed to create rabbitMQ client: %w", err)
	}
	defer rabbitClient.Close()

	grpcClient, err := grpc.NewClient(
		cfg.GRPCTarget,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer grpcClient.Close()

	resultCommitter := grpc_client.NewResultCommiter(grpcClient)

	processor := worker.NewProcessor(logger, cfg.DataDir, resultCommitter)

	consumer, err := rabbitmq.NewConsumer(rabbitClient, cfg.TaskQueueName)
	if err != nil {
		return fmt.Errorf("failed to create rabbitMQ consumer: %w", err)
	}

	consumerTag := uuid.New().String()
	deliveries, err := consumer.Consume(consumerTag)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		logger.Info("shutting down, stopping consumption...", "reason", ctx.Err())
		if err := consumer.Cancel(consumerTag); err != nil {
			logger.Error("failed to cancel consumer", "error", err)
		}
	}()

	logger.Info("worker started, waiting for messages...")

	for delivery := range deliveries {
		var msg task.Message
		if err := json.Unmarshal(delivery.Body, &msg); err != nil {
			logger.Error("failed to unmarshal task message", "error", err)
			if err := delivery.Nack(false, false); err != nil { // сообщение битое, не пересылать
				logger.Error("failed to acknowledge the delivery", "error", err)
			}
			continue
		}

		if err := processor.Process(context.Background(), msg); err != nil {
			logger.Error("failed to process task", "task_id", msg.TaskID, "error", err)
			if err := delivery.Nack(false, true); err != nil { // вернуть в очередь на повтор
				logger.Error("failed to acknowledge the delivery", "error", err)
			}
			continue
		}

		if err := delivery.Ack(false); err != nil {
			logger.Error("failed to ack delivery", "error", err)
		}
	}

	logger.Info("consumer channel closed, worker stopped")

	return nil
}
