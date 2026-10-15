package rabbitmq

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func (c *Client) DeclareTaskTopology(queueName string, retryDelay time.Duration) error {
	retryQueue := queueName + ".retry"
	retryExchange := queueName + ".retry.exchange"
	deadQueue := queueName + ".dead"

	// retry exchange
	if err := c.channel.ExchangeDeclare(
		retryExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare retry-exchange: %w", err)
	}

	// retry queue
	if _, err := c.channel.QueueDeclare(
		retryQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-message-ttl":             retryDelay.Milliseconds(),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": queueName,
		},
	); err != nil {
		return fmt.Errorf("declare retry-queue: %w", err)
	}

	// QueueBind: связывает retry-очередь с retry-exchange
	if err := c.channel.QueueBind(
		retryQueue,    // очередь назначения
		queueName,     // routing key
		retryExchange, // exchange
		false,
		nil,
	); err != nil {
		return fmt.Errorf("bind retry-exchange to retry-queue: %w", err)
	}

	// dead queue
	if _, err := c.channel.QueueDeclare(
		deadQueue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declare dead-queue: %w", err)
	}

	// main queue
	_, err := c.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    retryExchange,
			"x-dead-letter-routing-key": queueName,
		},
	)
	if err != nil {
		return fmt.Errorf("declare the message queue: %w", err)
	}

	return nil
}
