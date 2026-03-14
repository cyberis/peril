package pubsub

import (
	"encoding/json"
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

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T),
) error {

	ch, q, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var msg T
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}
			handler(msg)
			err = d.Ack(false)
			if err != nil {
				log.Printf("Error acknowledging message: %v", err)
			}
		}
	}()

	return nil
}
