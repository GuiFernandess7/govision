package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	channel *amqp.Channel
	queue   string
}

func NewRabbitMQPublisher(ch *amqp.Channel, queue string) *RabbitMQPublisher {
	return &RabbitMQPublisher{
		channel: ch,
		queue:   queue,
	}
}

func (p *RabbitMQPublisher) Publish(
	ctx context.Context,
	jobID string,
	imageURL string,
) error {
	body := fmt.Sprintf(`{"job_id": "%s", "image_url": "%s"}`, jobID, imageURL)
	return p.channel.PublishWithContext(
		ctx,
		"",
		p.queue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
			MessageId:   jobID,
			Timestamp:   time.Now(),
		},
	)
}
