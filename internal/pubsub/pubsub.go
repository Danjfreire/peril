package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	})

	return nil
}

func DeclateAndBind(
	conn *amqp.Connection,
	exchange string,
	queueName string,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("failed to create new channel")
	}

	durable := false
	autoDelete := false
	exclusive := false

	if queueType == Durable {
		durable = true
	} else {
		autoDelete = true
		exclusive = true
	}

	queue, err := ch.QueueDeclare(queueName, durable, autoDelete, exclusive, false, amqp.Table{"x-dead-letter-exchange": "peril_dlx"})
	if err != nil {
		log.Fatal("failed to declare queue")
	}

	ch.QueueBind(queue.Name, key, exchange, false, nil)

	return ch, queue, nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
	handler func(T) AckType,
) error {
	ch, queue, err := DeclateAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		log.Fatal("failed to declare and bind queue")
	}

	deliveryCh, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatal("failed to start delivering queue messages")
	}

	// go handleDelivery(deliveryCh, handler)
	go func() {
		defer ch.Close()
		for d := range deliveryCh {
			var msg T
			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				log.Println("failed to unmarshal message:", err)
				continue
			}

			handlerRes := handler(msg)
			switch handlerRes {
			case Ack:
				d.Ack(false)
				fmt.Println("Ack")
			case NackRequeue:
				d.Nack(false, true)
				fmt.Println("NackRequeue")
			case NackDiscard:
				d.Nack(false, false)
				fmt.Println("NackDiscard")
			}
		}

	}()

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange string, key string, val T) error {
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	err := enc.Encode(val)
	if err != nil {
		return err
	}

	ch.PublishWithContext(context.Background(), exchange, key, false, false, amqp.Publishing{
		ContentType: "application/gob",
		Body:        data.Bytes(),
	})

	return nil
}
