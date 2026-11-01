package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Sender struct {
	client    *Client
	queueName string
}

func NewSender(client *Client, queueName string, retryDelay time.Duration) (*Sender, error) {
	if err := client.DeclareTaskTopology(queueName, retryDelay); err != nil {
		return nil, fmt.Errorf("declare sender queue: %w", err)
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

	ch, err := s.client.connection.Channel()
	if err != nil {
		return fmt.Errorf("open AMQP channel: %w", err)
	}
	defer ch.Close()

	if err = ch.Confirm(false); err != nil {
		return fmt.Errorf("put channel into confirm mode: %w", err)
	}

	returns := ch.NotifyReturn(make(chan amqp.Return, 1))

	publishCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	confirmation, err := ch.PublishWithDeferredConfirmWithContext(
		publishCtx,
		"",
		s.queueName,
		true,
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

	acked, err := confirmation.WaitContext(publishCtx)
	if err != nil {
		return fmt.Errorf("wait confirmation: %w", err)
	}
	if !acked {
		return fmt.Errorf("broker did not confirm")
	}

	select {
	case returned, ok := <-returns:
		if !ok {
			return fmt.Errorf("publisher returns channel closed")
		}
		return fmt.Errorf(
			"Publication returned: %d %s",
			returned.ReplyCode,
			returned.ReplyText,
		)
	default:
		return nil
	}
}
