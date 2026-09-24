package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

func NewClient(amqpURL string) (*Client, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("open a RabbitMQ channel: %w", err)
	}

	return &Client{
		connection: conn,
		channel:    ch,
	}, nil
}

func (c *Client) Close() {
	c.channel.Close()
	c.connection.Close()
}
