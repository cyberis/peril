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

	// Create a new GameState for the user
	gameState := gamelogic.NewGameState(username)

	//Display the client help message
	gamelogic.PrintClientHelp()

	// Consume pause messages for this user
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.QueueTypeTransient, handlerPause(gameState))
	if err != nil {
		log.Fatalf("Error subscribing to pause messages: %v", err)
	}

	// Start a repl for supported commands
REPL:
	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "spawn":
			err = gameState.CommandSpawn(input)
			if err != nil {
				log.Printf("Error executing spawn command: %v", err)
			}
		case "move":
			_, err = gameState.CommandMove(input)
			if err != nil {
				log.Printf("Error executing move command: %v", err)
			}
		case "status":
			gameState.CommandStatus()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			break REPL
		default:
			log.Printf("Unknown command: %s. Type 'help' for a list of commands.", input[0])
		}
	}

	// wait for ctrl+c to quit
	fmt.Println("Press Ctrl+C to exit...")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("RabbitMQ connection closed.")

}
