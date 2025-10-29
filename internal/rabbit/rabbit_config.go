package rabbit

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dota2classic/d2c-go-models/util"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Rabbit struct {
	amqpURL  string
	exchange string
	Conn     *amqp.Connection
}

func NewRabbit(amqpURL string) *Rabbit {
	return &Rabbit{amqpURL: amqpURL, exchange: "app.events"}
}

var Instance *Rabbit

// Connect establishes the connection with auto-reconnect
func (r *Rabbit) Connect() error {
	for {
		conn, err := amqp.Dial(r.amqpURL)
		if err != nil {
			log.Printf("RabbitMQ connect failed: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		r.Conn = conn
		return nil
	}
}

// Channel creates a fresh channel, reconnecting if needed
func (r *Rabbit) getChannel() (*amqp.Channel, error) {
	for {
		if r.Conn == nil || r.Conn.IsClosed() {
			if err := r.Connect(); err != nil {
				continue
			}
		}

		ch, err := r.Conn.Channel()
		if err != nil {
			log.Printf("Failed to open channel: %v. Reconnecting...", err)
			r.Conn.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		return ch, nil
	}
}

func InitRabbit() {
	host := os.Getenv("RABBITMQ_HOST")
	port := util.GetEnvInt("RABBITMQ_PORT", 5672)

	username := os.Getenv("RABBITMQ_USER")
	password := os.Getenv("RABBITMQ_PASSWORD")

	exchange := "app.events"

	amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%d/", username, password, host, port)

	Instance = NewRabbit(amqpURL)

	ch, err := Instance.getChannel()
	if err != nil {
		Instance.Conn.Close()
		log.Fatalf("Failed to obtain channel %v", err)
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
		Instance.Conn.Close()
		log.Fatalf("Failed to create exchange %v", err)
	}

	Instance.initConsumers()

	log.Println("RabbitMQ consumer initialized")
}
