package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/cyberis/peril/internal/pubsub"
	"github.com/cyberis/peril/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server ... ")
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("Could not connect to RabbitMQ: %v", err)

	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Could not create channel: %v", err)
	}
	defer channel.Close()

	fmt.Println("Peril game server connected to RabbitMQ!")

	// Send a pause message immediately to test the connection
	err = pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
	if err != nil {
		log.Printf("Error publishing pause message: %v", err)
	} else {
		log.Printf("Published initial pause message to exchange '%s' with key '%s'", routing.ExchangePerilDirect, routing.PauseKey)
	}

	// wait for ctrl+c to quit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("RabbitMQ connection closed.")
}
