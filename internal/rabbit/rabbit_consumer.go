package rabbit

import (
	"context"
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/k8s"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dota2classic/d2c-go-models/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

var regions = []models.Region{
	models.REGION_RU_MOSCOW,
	models.REGION_RU_NOVOSIBIRSK,
	models.REGION_EU_CZECH,
}

func (r *Rabbit) initConsumers() {
	// Start multiple consumers
	for _, region := range regions {
		queue := fmt.Sprintf("d2c-gs-controller.LaunchGameServerCommand.%s", region)
		key := fmt.Sprintf("LaunchGameServerCommand.%s", region)

		r.startConsuming(queue, "app.events", key, func(msg []byte) error {
			var event models.LaunchGameServerCommand
			if err := json.Unmarshal(msg, &event); err != nil {
				return err
			}
			return handleLaunchGameServerCommand(&event)
		})
	}
}

// startConsuming starts a consumer for a given queue and handler
func (r *Rabbit) startConsuming(queue, exchange, key string, handler func(msg []byte) error) {
	go func() {
		for {
			ch, err := r.getChannel()
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}

			_, err = ch.QueueDeclare(queue, true, false, false, false, nil)
			if err != nil {
				log.Printf("Queue declare failed: %v", err)
				ch.Close()
				time.Sleep(2 * time.Second)
				continue
			}

			err = ch.QueueBind(queue, key, exchange, false, nil)
			if err != nil {
				log.Printf("Queue bind failed: %v", err)
				ch.Close()
				time.Sleep(2 * time.Second)
				continue
			}

			msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
			if err != nil {
				log.Printf("Consume failed: %v", err)
				ch.Close()
				time.Sleep(2 * time.Second)
				continue
			}

			for m := range msgs {
				if err := handler(m.Body); err != nil {
					// if its any error other than "job alreadty exists", we requeue
					shouldRequeue := !errors.Is(err, k8s.ErrJobAlreadyExists)
					m.Nack(false, shouldRequeue)
				} else {
					m.Ack(false)
				}
			}

			ch.Close()
			time.Sleep(2 * time.Second)
		}
	}()
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
		log.Printf("Failed to process message: %v", err)
		err = msg.Nack(false, true)
		if err != nil {
			log.Fatalf("Failed to nack message %v", err)
		}
	} else {
		err = msg.Ack(false)
		if err != nil {
			log.Fatalf("Failed to ack message %v", err)
		}
	}

}

func handleLaunchGameServerCommand(event *models.LaunchGameServerCommand) error {
	log.Printf("Launching game server for matchId %d", event.MatchID)
	mr, err := k8s.DeployMatchResources(context.Background(), k8s.GetClient(), event)
	if err != nil {
		log.Printf("Failed to deploy match: %v", err)
		return err
	}
	log.Printf("Match %d successfully deployed", event.MatchID)
	err = db.InsertMatchResources(db.MatchResources{
		MatchId:       event.MatchID,
		JobName:       mr.JobName,
		SecretName:    mr.SecretName,
		ConfigMapName: mr.ConfigMapName,
	})
	if err != nil {
		log.Printf("Failed to insert match: %v", err)
		return err
	}
	return nil
}
