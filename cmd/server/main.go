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

	// Subscribe to the game logs topic with a durable queue so we can see the history of games when we restart the server
	err = pubsub.SubscribeGob(conn, routing.ExchangePerilTopic, routing.GameLogSlug, routing.GameLogsKey, pubsub.QueueTypeDurable, handlerGameLog())
	if err != nil {
		log.Fatalf("Could not subscribe to game logs: %v", err)
	}

	// Print the game server help function
	gamelogic.PrintServerHelp()

	// Start REPL
REPL:
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "help":
			gamelogic.PrintServerHelp()
		case "pause":
			err := pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
			if err != nil {
				log.Printf("Error publishing pause message: %v", err)
			} else {
				log.Printf("Published pause message to exchange '%s' with key '%s'", routing.ExchangePerilDirect, routing.PauseKey)
			}
		case "resume":
			err := pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
			if err != nil {
				log.Printf("Error publishing resume message: %v", err)
			} else {
				log.Printf("Published resume message to exchange '%s' with key '%s'", routing.ExchangePerilDirect, routing.PauseKey)
			}
		case "quit":
			log.Printf("Shutting down Peril server...")
			break REPL

		default:
			log.Printf("Unknown command: %s. Type 'help' for a list of commands.", input)
		}
	}

	// wait for ctrl+c to quit
	fmt.Println("Press Ctrl+C to exit...")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("RabbitMQ connection closed.")
}
