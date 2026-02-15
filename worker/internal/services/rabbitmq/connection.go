package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbittMQConnection(host string, port string, username string, password string) (*amqp.Connection, error) {
	connString := fmt.Sprintf(
		"amqp://%v:%v@%v:%v/",
		username,
		password,
		host,
		port,
	)
	conn, err := amqp.Dial(connString)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
