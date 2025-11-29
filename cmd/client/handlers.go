package main

import (
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gameState *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(state routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gameState.HandlePause(state)
		return pubsub.Ack
	}
}

func handlerMove(gameState *gamelogic.GameState, publishChannel *amqp.Channel) func(move gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		moveOutcome := gameState.HandleMove(move)
		switch moveOutcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				publishChannel,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gameState.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gameState.GetPlayerSnap(),
				},
			)

			if err != nil {
				fmt.Printf("error: %s", err)
				return pubsub.NackRequeue
			}

			return pubsub.Ack
		}

		fmt.Println("Unknown move outcome")
		return pubsub.NackDiscard
	}
}

func handlerWar(gamestate *gamelogic.GameState, ch *amqp.Channel) func(dw gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(dw gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		warOutcome, winner, loser := gamestate.HandleWar(dw)

		switch warOutcome {
		case gamelogic.WarOutcomeDraw:
			ackRes := routing.PublishGameLog(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+dw.Attacker.Username, routing.GameLog{
				CurrentTime: time.Now(),
				Message:     fmt.Sprintf("A war between %v and %v resulted in a draw", winner, loser),
			})
			return ackRes
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeOpponentWon:
			ackRes := routing.PublishGameLog(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+dw.Attacker.Username, routing.GameLog{
				CurrentTime: time.Now(),
				Message:     fmt.Sprintf("%v won a war agains %v", winner, loser),
			})
			return ackRes
		case gamelogic.WarOutcomeYouWon:
			ackRes := routing.PublishGameLog(ch, routing.ExchangePerilTopic, routing.GameLogSlug+"."+dw.Attacker.Username, routing.GameLog{
				CurrentTime: time.Now(),
				Message:     fmt.Sprintf("%v won a war agains %v", winner, loser),
			})
			return ackRes
		}

		fmt.Println("Unknown war outcome")
		return pubsub.NackDiscard
	}
}
