package pubsub

import (
	"bytes"
	"encoding/gob"
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

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

const (
	DeadLetterExchangeHeader = "x-dead-letter-exchange"
	DeadLetterExchangeName   = "peril_dlx"
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
	args := amqp.Table{
		DeadLetterExchangeHeader: DeadLetterExchangeName,
	}

	q, err := ch.QueueDeclare(
		queueName,  // name
		durable,    // durable
		autoDelete, // delete when unused
		exclusive,  // exclusive
		false,      // no-wait
		args,       // arguments
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
	handler func(T) AckType, // a handler function that processes the message and returns an acktype indicating how to acknowledge the message
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
		defer ch.Close()
		for d := range msgs {
			var msg T
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("Error unmarshalling message: %v", err)
				continue
			}
			ackType := handler(msg)
			switch ackType {
			case Ack:
				err = d.Ack(false)
				if err != nil {
					log.Printf("Error acknowledging message: %v", err)
				}
				log.Printf("Acknowledged message with routing key '%s': %s", d.RoutingKey, string(d.Body))
			case NackRequeue:
				err = d.Nack(false, true)
				if err != nil {
					log.Printf("Error nack'ing message for requeue: %v", err)
				}
				log.Printf("Nack'ed message for requeue with routing key '%s': %s", d.RoutingKey, string(d.Body))
			case NackDiscard:
				err = d.Nack(false, false)
				if err != nil {
					log.Printf("Error nack'ing message for discard: %v", err)
				}
				log.Printf("Nack'ed message for discard with routing key '%s': %s", d.RoutingKey, string(d.Body))
			default:
				log.Printf("Unknown AckType returned by handler: %v. Message with routing key '%s' was not acknowledged.", ackType, d.RoutingKey)
			}
		}
	}()

	return nil
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType, // a handler function that processes the message and returns an acktype indicating how to acknowledge the message
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
		defer ch.Close()
		for d := range msgs {
			var msg T
			buf := bytes.NewBuffer(d.Body)
			decoder := gob.NewDecoder(buf)
			if err := decoder.Decode(&msg); err != nil {
				log.Printf("Error decoding message: %v", err)
				continue
			}
			ackType := handler(msg)
			switch ackType {
			case Ack:
				err = d.Ack(false)
				if err != nil {
					log.Printf("Error acknowledging message: %v", err)
				}
				log.Printf("Acknowledged message with routing key '%s': %s", d.RoutingKey, string(d.Body))
			case NackRequeue:
				err = d.Nack(false, true)
				if err != nil {
					log.Printf("Error nack'ing message for requeue: %v", err)
				}
				log.Printf("Nack'ed message for requeue with routing key '%s': %s", d.RoutingKey, string(d.Body))
			case NackDiscard:
				err = d.Nack(false, false)
				if err != nil {
					log.Printf("Error nack'ing message for discard: %v", err)
				}
				log.Printf("Nack'ed message for discard with routing key '%s': %s", d.RoutingKey, string(d.Body))
			default:
				log.Printf("Unknown AckType returned by handler: %v. Message with routing key '%s' was not acknowledged.", ackType, d.RoutingKey)
			}
		}
	}()

	return nil
}
