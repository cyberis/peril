package main

import (
	"fmt"
	"log"

	"github.com/cyberis/peril/internal/gamelogic"
	"github.com/cyberis/peril/internal/pubsub"
	"github.com/cyberis/peril/internal/routing"
)

func handlerGameLog() func(routing.GameLog) pubsub.AckType {
	return func(gameLog routing.GameLog) pubsub.AckType {
		defer fmt.Print("> ")
		err := gamelogic.WriteLog(gameLog)
		if err != nil {
			log.Printf("Error writing game log to disk: %v", err)
			return pubsub.NackRequeue // This could be due to a transient error, so we requeue to try again
		}
		return pubsub.Ack
	}
}
