package rabbit

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/dota2classic/d2c-go-models/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

func initConsumer(ch *amqp.Channel) {
	initQueue(ch, models.REGION_RU_MOSCOW)
	initQueue(ch, models.REGION_RU_NOVOSIBIRSK)
	initQueue(ch, models.REGION_EU_CZECH)
}

func initQueue(ch *amqp.Channel, region models.Region) {
	serviceName := "d2c-gs-controller"
	messageName := "LaunchGameServerCommand"
	queueName := fmt.Sprintf("%s.%s.%s", serviceName, messageName, region)

	// Ensure queue exists
	_, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Start consuming
	msgs, err := ch.Consume(
		queueName, // queue
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("failed to register consumer: %v", err)
	}

	log.Printf("Start consuming queue %s", queueName)

	// 5. Consume messages (infinite loop)
	for msg := range msgs {
		log.Printf("Received: %s", msg.Body)
		handleMessage(&msg, handleLaunchGameServerCommand)
	}
}

func handleMessage[T any](msg *amqp.Delivery, handler func(event *T) error) {
	var event T
	err := json.Unmarshal(msg.Body, &event)
	if err != nil {
		log.Printf("Failed to unmarshal message, nacking: %v", err)
		err := msg.Nack(false, false)
		if err != nil {
			log.Fatalf("Failed to nack message %v", err)
		}
	}

	err = handler(&event)

	if err != nil {
		err = msg.Ack(false)
		if err != nil {
			log.Fatalf("Failed to ack message %v", err)
		}
	} else {
		log.Printf("Failed to process message: %v", err)
		err = msg.Nack(false, true)
		if err != nil {
			log.Fatalf("Failed to nack message %v", err)
		}
	}

}

func handleLaunchGameServerCommand(event *models.LaunchGameServerCommand) error {
	log.Println("Launching game server YASSSS")
	return nil
}
