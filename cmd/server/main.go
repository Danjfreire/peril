package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	connectionString := "amqp://guest:guest@localhost:5672/"
	connection, err := amqp.Dial(connectionString)
	if err != nil {
		log.Fatal("Failed to connect to RabbitMQ")
	}

	defer connection.Close()
	log.Print("Connection started with RabbitMQ")

	channel, err := connection.Channel()
	if err != nil {
		log.Fatal("Failed to open a channel")
	}

	defer channel.Close()

	_, _, err = pubsub.DeclateAndBind(connection, "peril_topic", "game_logs", "game_logs.*", pubsub.Durable)
	if err != nil {
		log.Fatal("Failed to declare and bind game logs queue:", err)
	}

	gamelogic.PrintServerHelp()

	for {
		input := gamelogic.GetInput()

		if len(input) == 0 {
			continue
		}

		if input[0] == "pause" {
			log.Println("Sending pause message")
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
			continue
		}

		if input[0] == "resume" {
			log.Println("Sending resume message")
			pubsub.PublishJSON(channel, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
			continue
		}

		if input[0] == "exit" {
			log.Println("Exiting server")
			break
		}

		log.Println("Unknown command:", input[0])
	}

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

	log.Print("Program exiting...")
}
