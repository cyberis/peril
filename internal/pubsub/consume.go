package pubsub

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Create the SimpleQueueType enum for durable vs transient queues
type SimpleQueueType int

const (
	QueueTypeDurable SimpleQueueType = iota
	QueueTypeTransient
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // SimpleQueueType is an "enum" type I made to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	// Declare the Queue
	durable := queueType == QueueTypeDurable
	autoDelete := queueType == QueueTypeTransient
	exclusive := queueType == QueueTypeTransient

	q, err := ch.QueueDeclare(
		queueName,  // name
		durable,    // durable
		autoDelete, // delete when unused
		exclusive,  // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	// Bind the Queue to the Exchange with the routing key
	err = ch.QueueBind(
		q.Name,   // queue name
		key,      // routing key
		exchange, // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	log.Printf("Declared and bound queue '%s' to exchange '%s' with key '%s'", q.Name, exchange, key)
	return ch, q, nil
}
