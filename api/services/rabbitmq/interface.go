package rabbitmq

import "context"

type JobPublisher interface {
	Publish(ctc context.Context, jobID string, imageURL string) error
}