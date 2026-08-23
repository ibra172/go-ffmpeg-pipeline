package queue

import "github.com/rabbitmq/amqp091-go"

var (
	rabbitAddr = "amqp://guest:guest@127.0.0.1:5672/"

	rabbitConn *amqp.Connection

	rabbitChan *amqp.Channel
)
д