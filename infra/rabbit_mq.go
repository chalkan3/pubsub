package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/badico-cloud-hub/pubsub/dto"
	amqp "github.com/rabbitmq/amqp091-go"
)

//RabbitMQ is struct for broker in amazon mq
type RabbitMQ struct {
	url      string
	conn     *amqp.Connection
	ch       *amqp.Channel
	queue    amqp.Queue
	queueDlq amqp.Queue
}

//NewRabbitMQ return new instance of RabbitMQ
func NewRabbitMQ() *RabbitMQ {

	amzMqUrl := os.Getenv("AMAZON_MQ_URL")
	return &RabbitMQ{
		url: amzMqUrl,
	}
}

//Setup is configure broker
func (r *RabbitMQ) Setup() error {
	connection, err := amqp.Dial(r.url)
	if err != nil {
		return err
	}
	channel, err := connection.Channel()
	if err != nil {
		return err
	}
	q, err := channel.QueueDeclare(
		"pubsub", // name
		true,     // durable
		false,    // delete when unused
		false,    // exclusive
		false,    // no-wait
		nil,      // arguments
	)

	if err != nil {
		return err
	}

	qd, err := channel.QueueDeclare(
		"pubsub_dlq", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return err
	}

	r.conn = connection
	r.ch = channel
	r.queue = q
	r.queueDlq = qd
	return nil
}

func (r *RabbitMQ) Ack(deliveredTag uint64) error {
	err := r.ch.Ack(deliveredTag, false) // <-- Difference
	if err != nil {
		return err
	}
	return nil
}

func (r *RabbitMQ) NumberOfMessagesQueue() error {
	fmt.Printf("%s: contains -> %v\n", r.queue.Name, r.queue.Messages)
	fmt.Printf("%s: contains -> %v\n", r.queueDlq.Name, r.queueDlq.Messages)
	return nil
}

//Producer is send message to broker
func (r *RabbitMQ) Producer(queueMessage dto.QueueMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	queueMessageBytes, err := json.Marshal(queueMessage)
	if err != nil {
		return err
	}
	if err := r.ch.PublishWithContext(
		ctx,
		"",
		r.queue.Name,
		false,
		false,
		amqp.Publishing{
			Timestamp: time.Now(),
			Type:      "text/plain",
			Body:      queueMessageBytes,
		}); err != nil {
		return err
	}

	return nil
}

//Dlq is send message to dlq broker
func (r *RabbitMQ) Dlq(queueMessage dto.QueueMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	queueMessageBytes, err := json.Marshal(queueMessage)
	if err != nil {
		return err
	}
	if err := r.ch.PublishWithContext(
		ctx,
		"",
		r.queueDlq.Name,
		false,
		false,
		amqp.Publishing{
			Timestamp: time.Now(),
			Type:      "text/plain",
			Body:      queueMessageBytes,
		}); err != nil {
		return err
	}

	return nil
}

//Consumer is return channel for consume from broker
func (r *RabbitMQ) Consumer() (<-chan amqp.Delivery, error) {
	msgs, err := r.ch.Consume(
		r.queue.Name, // queue
		"",           // consumer
		true,         // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

//ConsumerDlq is return channel for consume dlq from broker
func (r *RabbitMQ) ConsumerDlq() (<-chan amqp.Delivery, error) {
	msgs, err := r.ch.Consume(
		r.queueDlq.Name, // queue
		"",              // consumer
		true,            // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

//Release is close connection
func (r *RabbitMQ) Release() {
	r.conn.Close()
}
