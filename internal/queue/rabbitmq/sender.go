package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Sender struct {
	client    *Client
	queueName string
}

func NewSender(client *Client, queueName string) (*Sender, error) {
	_, err := client.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("declare a message queue: %w", err)
	}

	return &Sender{
		client:    client,
		queueName: queueName,
	}, nil
}

func (s *Sender) Send(ctx context.Context, msg task.Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal a task message: %w", err)
	}

	err = s.client.channel.PublishWithContext(
		ctx,
		"",
		s.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish a task message: %w", err)
	}

	return nil
}
