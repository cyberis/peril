package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/cyberis/peril/internal/gamelogic"
	"github.com/cyberis/peril/internal/pubsub"
	"github.com/cyberis/peril/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("Could not connect to RabbitMQ: %v", err)

	}
	defer conn.Close()

	//Ask the user for their username via the ClientWelcome function
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Error during client welcome: %v", err)
	}
	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)

	// Decare and bind a queue for the user, and subscribe to the pause messages
	ch, q, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.QueueTypeTransient)
	if err != nil {
		log.Fatalf("Error declaring and binding queue '%s': %v", queueName, err)
	}
	log.Printf("Declared and bound queue '%s' for user '%s'", q.Name, username)
	defer ch.Close()

	// wait for ctrl+c to quit
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("RabbitMQ connection closed.")

}
