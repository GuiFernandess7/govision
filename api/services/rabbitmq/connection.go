package rabbitmq

import (
	"log"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublisherFactory() JobPublisher {
	log.Println("[INIT] - Connecting to RabbitMQ")
	rabbitmq_url := os.Getenv("RABBITMQ_URL")
	queue := os.Getenv("RABBITMQ_QUEUE")

	if rabbitmq_url == "" || queue == "" {
		log.Fatalf("[ERROR] - Environment variables not found.")
	}

	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("[ERROR] - Error connecting to RabbittMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("[ERROR] - Error getting RabbitMQ channel: %v", err)
	}

	_, err = ch.QueueDeclare(
		queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[ERROR] - Error declaring queue: %v", err)
	}

	return NewRabbitMQPublisher(ch, queue)
}
