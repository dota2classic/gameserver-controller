package rabbit

import (
	"context"
	"d2c-gs-controller/internal/util"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitContainer struct {
	Conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

var Client *rabbitContainer

func InitRabbitPublisher() {
	host := os.Getenv("RABBITMQ_HOST")
	port := util.GetEnvInt("RABBITMQ_PORT", 5672)

	username := os.Getenv("RABBITMQ_USER")
	password := os.Getenv("RABBITMQ_PASSWORD")

	exchange := "app.events"

	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%d/", username, password, host, port)

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatal(err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		log.Fatal(err)
	}

	// Declare our exchange
	err = ch.ExchangeDeclare(
		exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		log.Fatal(err)
	}

	initConsumer(ch)

	Client = &rabbitContainer{
		Conn:     conn,
		channel:  ch,
		exchange: exchange,
	}

	log.Println("RabbitMQ publisher initialized")
}

func publishWithRetry[T any](event *T, routingKey string, retries int) error {
	var err error

	message, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("can't serialize payload: %w", err)
	}

	for attempt := 1; attempt <= retries; attempt++ {
		err = Client.channel.Publish(
			Client.exchange, // exchange
			routingKey,
			false, // mandatory
			false, // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        message,
				Timestamp:   time.Now(),
			},
		)

		if err == nil {
			return nil
		}

		// Check if it's a transient error (network, closed Conn, etc.)
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("rabbitmq publish failed (attempt %d/%d): %v — retrying...\n", attempt, retries, err)
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
			continue
		}

		// Non-retryable error
		return fmt.Errorf("rabbitmq publish failed: %w", err)
	}

	return fmt.Errorf("rabbitmq publish failed after %d retries: %w", retries, err)
}

// Close the connection and channel
func (r *rabbitContainer) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}
