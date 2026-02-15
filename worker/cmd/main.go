package main

import (
	"errors"
	"fmt"
	sabbitmq "govision_worker/internal/services/rabbitmq"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	_ = godotenv.Load()
	rabbitConnString := os.Getenv("RABBITMQ_URL")
	rabbitQueueString := os.Getenv("RABBITMQ_QUEUE")

	if rabbitConnString == "" || rabbitQueueString == "" {
		log.Printf("[ERROR] - Environment variables no found.")
		panic(errors.New("Environment variables not found."))
	}

	rabbitMQConnection, err := sabbitmq.NewRabbittMQConnection(rabbitConnString)
	if err != nil {
		log.Printf("[ERROR] - RabbitMQ connection error: %v", err)
		panic(err)
	}
	defer rabbitMQConnection.Close()

	ch, err := rabbitMQConnection.Channel()
	if err != nil {
		log.Printf("[ERROR] - RabbitMQ channel error: %v", err)
		panic(err)
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		rabbitQueueString,
		"",
		true,
		false,
		false,
		false,
		nil,
	)

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			fmt.Printf("Received message: %s\n", d.Body)
		}
	}()

	fmt.Println("Successfully connected to RabbitMQ instance")
	fmt.Println("[*] - Waiting for messages")
	<-forever
}
