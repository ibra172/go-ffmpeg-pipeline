package rabbitmq

import (
	"context"
	"fmt"
	"maps"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	client     *Client
	queueName  string
	maxRetries int64
	returns    <-chan amqp.Return
}

func NewConsumer(client *Client, queueName string, retryDelay time.Duration, maxRetries int64) (*Consumer, error) {
	if maxRetries < 0 {
		return nil, fmt.Errorf("maxRetries must be greater than or equal to 0")
	}

	if err := client.DeclareTaskTopology(queueName, retryDelay); err != nil {
		return nil, fmt.Errorf("declare consumer queue: %w", err)
	}

	err := client.channel.Qos(
		1,     // prefetchCount
		0,     // prefetchSize
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("set QoS: %w", err)
	}

	if err := client.channel.Confirm(false); err != nil {
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	returns := client.channel.NotifyReturn(make(chan amqp.Return, 1))

	return &Consumer{
		client:     client,
		queueName:  queueName,
		maxRetries: maxRetries,
		returns:    returns,
	}, nil
}

func (c *Consumer) Consume(consumerTag string) (<-chan amqp.Delivery, error) {
	deliveries, err := c.client.channel.Consume(
		c.queueName,
		consumerTag, // consumer tag
		false,       // autoAck
		false,       // exclusive
		false,       // noLocal
		false,       // noWait
		nil,         // args
	)
	if err != nil {
		return nil, fmt.Errorf("start consuming: %w", err)
	}

	return deliveries, nil
}

func (c *Consumer) Cancel(consumerTag string) error {
	return c.client.channel.Cancel(consumerTag, false)
}

func (c *Consumer) RetryOrDead(ctx context.Context, delivery amqp.Delivery, cause error) error {
	count, countErr := c.retryCount(delivery.Headers)
	if countErr == nil && count < c.maxRetries {
		return delivery.Nack(false, false)
	}
	reason := "retry limit reached"
	if countErr != nil {
		reason = "invalid retry headers: " + countErr.Error()
	}

	return c.Dead(ctx, delivery, cause.Error()+"; "+reason)
}

func (c *Consumer) retryCount(headers amqp.Table) (int64, error) {
	xDeath, ok := headers["x-death"]
	if !ok {
		return 0, nil
	}
	entries, ok := xDeath.([]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected x-death type: %T", xDeath)
	}

	var count int64
	for _, value := range entries {
		entry, ok := value.(amqp.Table)
		if !ok {
			return 0, fmt.Errorf("unexpected x-death entry type: %T", value)
		}

		queue, ok := entry["queue"].(string)
		if !ok {
			return 0, fmt.Errorf("invalid queue field in x-death")
		}
		if queue != c.queueName {
			continue
		}

		reason, ok := entry["reason"].(string)
		if !ok {
			return 0, fmt.Errorf("invalid reason field in x-death")
		}
		if reason != "rejected" {
			continue
		}

		count, ok = entry["count"].(int64)
		if !ok {
			return 0, fmt.Errorf("invalid count field in x-death")
		}
		if count < 0 {
			return 0, fmt.Errorf("count can not be < 0")
		}
		break
	}

	return count, nil
}

func (c *Consumer) Dead(ctx context.Context, delivery amqp.Delivery, reason string) error {
	deadCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second*2)
	defer cancel()

	if err := c.publishToDead(deadCtx, delivery, reason); err != nil {
		return fmt.Errorf("publish message to dead-queue: %w", err)
	}

	if err := delivery.Ack(false); err != nil {
		return fmt.Errorf("ack delivery: %w", err)
	}

	return nil
}

func (c *Consumer) publishToDead(ctx context.Context, delivery amqp.Delivery, reason string) error {
	headers := make(amqp.Table, len(delivery.Headers)+1)
	maps.Copy(headers, delivery.Headers)
	headers["x-dead-reason"] = reason

	confirmation, err := c.client.channel.PublishWithDeferredConfirmWithContext(
		ctx,
		"",
		c.queueName+".dead",
		true,  // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  delivery.ContentType,
			DeliveryMode: amqp.Persistent,
			Body:         delivery.Body,
			Headers:      headers,
		},
	)
	if err != nil {
		return fmt.Errorf("publishing: %w", err)
	}

	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("wait publisher confirmation: %w", err)
	}
	if !acked {
		return fmt.Errorf("broker did not confirm DLQ publication")
	}

	select {
	case returned, ok := <-c.returns:
		if !ok {
			return fmt.Errorf("publisher returns channel closed")
		}
		return fmt.Errorf(
			"DLQ publication returned: %d %s",
			returned.ReplyCode,
			returned.ReplyText,
		)
	default:
		return nil
	}
}
