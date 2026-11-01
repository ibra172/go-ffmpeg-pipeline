package rabbitmq

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
)

func TestSenderIntegration(t *testing.T) {
	rabbitmqTestURL := os.Getenv("RABBITMQ_TEST_URL")
	if rabbitmqTestURL == "" {
		t.Skip("RABBITMQ_TEST_URL is not set")
	}

	client, err := NewClient(rabbitmqTestURL)
	if err != nil {
		t.Fatalf("failed to create rabbitMQ client: %v", err)
	}
	t.Cleanup(client.Close)

	t.Run("existing queue", func(t *testing.T) {
		queue, err := client.channel.QueueDeclare(
			"test.sender."+uuid.NewString(),
			false, // durable
			false, // autoDelete
			true,  // exclusive: удалится при закрытии соединения
			false, // noWait
			nil,
		)
		if err != nil {
			t.Fatalf("declare queue: %v", err)
		}

		sender := &Sender{client: client, queueName: queue.Name}

		taskMsg := task.Message{TaskID: uuid.New()}
		err = sender.Send(context.Background(), taskMsg)
		if err != nil {
			t.Fatalf("error = %v, want nil", err)
		}

		msg, ok, err := client.channel.Get(queue.Name, true)
		if err != nil {
			t.Fatalf("receive delivery from queue: %v", err)
		}
		if !ok {
			t.Fatalf("no messages in the queue")
		}

		var message task.Message
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			t.Fatalf("unmarshall message: %v", err)
		}
		if message.TaskID != taskMsg.TaskID {
			t.Fatalf("expected task id %s, got %s", taskMsg.TaskID, message.TaskID)
		}
	})

	t.Run("missing queue", func(t *testing.T) {
		sender := &Sender{
			client:    client,
			queueName: "test.sender.missing." + uuid.NewString(),
		}

		err := sender.Send(context.Background(), task.Message{TaskID: uuid.New()})
		if err == nil {
			t.Fatalf("expected non-nil error, but got nil")
		}

		if !strings.Contains(err.Error(), "NO_ROUTE") {
			t.Fatalf("error = %v, expected 'NO_ROUTE' in error", err)
		}

	})
}
