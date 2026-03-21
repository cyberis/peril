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

	publishChannel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Could not create channel: %v", err)
	}
	defer publishChannel.Close()

	//Ask the user for their username via the ClientWelcome function
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Error during client welcome: %v", err)
	}
	queueNamePause := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	queueNameMove := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)

	// Create a new GameState for the user
	gameState := gamelogic.NewGameState(username)

	//Display the client help message
	gamelogic.PrintClientHelp()

	// Consume pause messages for this user
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queueNamePause, routing.PauseKey, pubsub.QueueTypeTransient, handlerPause(gameState))
	if err != nil {
		log.Fatalf("Error subscribing to pause messages: %v", err)
	}

	// Consume move messages for this user
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, queueNameMove, routing.ArmyMovesKey, pubsub.QueueTypeTransient, handlerMove(gameState, publishChannel))
	if err != nil {
		log.Fatalf("Error subscribing to move messages: %v", err)
	}

	// Consume all war recognition messages for any user -- this is dumb but is part of the exercise
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, routing.WarRecognitionsPrefix, fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, "*"), pubsub.QueueTypeDurable, handlerWar(gameState))
	if err != nil {
		log.Fatalf("Error subscribing to war recognition messages: %v", err)
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
			mv, err := gameState.CommandMove(input)
			if err != nil {
				log.Printf("Error executing move command: %v", err)
				continue
			}
			err = pubsub.PublishJSON(publishChannel, routing.ExchangePerilTopic, routing.ArmyMovesKey, mv)
			if err != nil {
				log.Printf("Error publishing move command: %v", err)
				continue
			}
			log.Printf("Published move command to exchange '%s' with key '%s': %+v", routing.ExchangePerilTopic, routing.ArmyMovesKey, mv)
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
