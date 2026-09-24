package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	client    *Client
	queueName string
}

func NewConsumer(client *Client, queueName string) (*Consumer, error) {
	_, err := client.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("declare the message queue: %w", err)
	}

	err = client.channel.Qos(
		1,     // prefetchCount
		0,     // prefetchSize
		false, // global
	)
	if err != nil {
		return nil, fmt.Errorf("set QoS: %w", err)
	}

	return &Consumer{
		client:    client,
		queueName: queueName,
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
