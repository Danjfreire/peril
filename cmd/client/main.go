package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connectionString := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatal("Client failed to connect to RabbitMQ")
	}

	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		log.Fatal("Failed to start client channel")
	}

	defer channel.Close()

	userName, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatal("failed to collect username")
	}

	gameState := gamelogic.NewGameState(userName)
	pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilDirect,
		fmt.Sprintf("pause.%v", userName),
		routing.PauseKey,
		pubsub.Transient,
		handlerPause(gameState),
	)
	pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		fmt.Sprintf("army_moves.%v", userName),
		"army_moves.*",
		pubsub.Transient,
		handlerMove(gameState, channel),
	)
	pubsub.SubscribeJSON(
		connection,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.Durable,
		handlerWar(gameState, channel),
	)

	for {
		input := gamelogic.GetInput()

		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "spawn":
			gameState.CommandSpawn(input)
		case "move":
			armyMove, err := gameState.CommandMove(input)
			if err != nil {
				log.Println("Error moving:", err)
			} else {
				pubsub.PublishJSON(channel, routing.ExchangePerilTopic, "army_moves."+userName, armyMove)
				log.Println("Move successful")
			}

		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			log.Println("Spamming not allowed yet")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			continue
		}
	}
}
