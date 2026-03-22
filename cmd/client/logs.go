package main

import (
	"fmt"
	"time"

	"github.com/cyberis/peril/internal/pubsub"
	"github.com/cyberis/peril/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func publishGameLog(ch *amqp.Channel, username string, msg string) error {
	key := fmt.Sprintf("%s.%s", routing.GameLogSlug, username)
	gameLog := routing.GameLog{
		CurrentTime: time.Now(),
		Message:     msg,
		Username:    username,
	}
	return pubsub.PublishGob(ch, routing.ExchangePerilTopic, key, gameLog)
}
