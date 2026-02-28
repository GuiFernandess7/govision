package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"govision/worker/internal/repository/postgres"
	"govision/worker/internal/services/rabbitmq"
	"govision/worker/internal/services/roboflow"
	"govision/worker/internal/worker"

	pgconn "govision/worker/internal/services/postgres"

	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	_ = godotenv.Load()
	rabbitConnString := os.Getenv("RABBITMQ_URL")
	rabbitQueueString := os.Getenv("RABBITMQ_QUEUE")
	roboflowAPIKey := os.Getenv("ROBOFLOW_API_KEY")
	roboflowWorkspaceID := os.Getenv("ROBOFLOW_WORKSPACE_ID")
	roboflowWorkflowID := os.Getenv("ROBOFLOW_WORKFLOW_ID")
	databaseURL := os.Getenv("DATABASE_URL")

	if rabbitConnString == "" || rabbitQueueString == "" {
		log.Printf("[ERROR] - Environment variables not found.")
		panic(errors.New("environment variables not found"))
	}

	if roboflowAPIKey == "" || roboflowWorkspaceID == "" || roboflowWorkflowID == "" {
		log.Printf("[ERROR] - Roboflow environment variables not found.")
		panic(errors.New("ROBOFLOW_API_KEY, ROBOFLOW_WORKSPACE_ID and ROBOFLOW_WORKFLOW_ID must be set"))
	}

	if databaseURL == "" {
		log.Printf("[ERROR] - DATABASE_URL environment variable not found.")
		panic(errors.New("DATABASE_URL must be set"))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// PostgreSQL connection via GORM
	db, err := pgconn.NewConnection(databaseURL)
	if err != nil {
		log.Printf("[ERROR] - PostgreSQL connection error: %v", err)
		panic(err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Repository
	predictionRepo := postgres.NewPredictionRepository(db)

	// RabbitMQ connection
	rabbitMQConnection, err := rabbitmq.NewRabbittMQConnection(rabbitConnString)
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

	// Consumer
	consumer := rabbitmq.NewRabbitMQConsumer(ch, rabbitQueueString)

	msgs, err := consumer.Consume(ctx)
	if err != nil {
		log.Printf("[ERROR] - Failed to start consuming: %v", err)
		panic(err)
	}

	// Roboflow client
	rfClient := roboflow.NewClient(roboflowAPIKey, roboflowWorkspaceID, roboflowWorkflowID)

	// Worker
	w := worker.New(rfClient, predictionRepo)

	fmt.Println("Successfully connected to RabbitMQ instance")
	fmt.Println("[*] - Waiting for messages")

	w.ProcessMessages(ctx, msgs)
}
