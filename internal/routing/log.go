package routing

import (
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/rabbitmq/amqp091-go"
)

func PublishGameLog(ch *amqp091.Channel, exchange string, key string, log GameLog) pubsub.AckType {
	err := pubsub.PublishGob(ch, exchange, key, log)
	if err != nil {
		return pubsub.NackRequeue
	}

	return pubsub.Ack
}
