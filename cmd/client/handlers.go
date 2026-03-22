package main

import (
	"fmt"
	"log"

	"github.com/cyberis/peril/internal/gamelogic"
	"github.com/cyberis/peril/internal/pubsub"
	"github.com/cyberis/peril/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, publishCh *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		moveOutcome := gs.HandleMove(move)
		switch moveOutcome {
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			key := fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, gs.GetUsername())
			war := gamelogic.RecognitionOfWar{
				Attacker: move.Player,
				Defender: gs.GetPlayerSnap(),
			}
			err := pubsub.PublishJSON(publishCh, routing.ExchangePerilTopic, key, war)
			if err != nil {
				log.Printf("Error publishing war recognition: %v", err)
				return pubsub.NackRequeue // This could be due to a transient error, so we requeue to try again
			}
			return pubsub.Ack // We ack the move message regardless of the outcome of the war recognition publish, because the move itself was processed successfully
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerWar(gs *gamelogic.GameState, publishCh *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(recog gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		warOutcome, winner, loser := gs.HandleWar(recog)
		switch warOutcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue // Allow other user to handle this message
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			outcomeMessage := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishGameLog(publishCh, gs.GetUsername(), outcomeMessage)
			if err != nil {
				log.Printf("Error publishing game log: %v", err)
				return pubsub.NackRequeue // This could be due to a transient error, so we requeue to try again
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			outcomeMessage := fmt.Sprintf("%s won a war against %s", winner, loser)
			err := publishGameLog(publishCh, gs.GetUsername(), outcomeMessage)
			if err != nil {
				log.Printf("Error publishing game log: %v", err)
				return pubsub.NackRequeue // This could be due to a transient error, so we requeue to try again
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			outcomeMessage := fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			err := publishGameLog(publishCh, gs.GetUsername(), outcomeMessage)
			if err != nil {
				log.Printf("Error publishing game log: %v", err)
				return pubsub.NackRequeue // This could be due to a transient error, so we requeue to try again
			}
			return pubsub.Ack
		default:
			fmt.Println("error: unknown war outcome")
			return pubsub.NackDiscard
		}
	}
}
