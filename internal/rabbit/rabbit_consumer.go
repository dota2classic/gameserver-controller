package rabbit

import (
	"d2c-gs-controller/internal/k8s"
	"d2c-gs-controller/internal/rabbit/queues"
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
		key := fmt.Sprintf("LaunchGameServerCommand.%s", region)

		r.startConsuming(fmt.Sprintf("d2c-gs-controller.LaunchGameServerCommand.%s", region), Exchange, key, 10, func(msg *amqp.Delivery) error {
			var event models.LaunchGameServerCommand
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				return err
			}
			return queues.HandleLaunchGameServerCommand(&event)
		})
	}
}

// startConsuming starts a consumer for a given queue and handler
func (r *Rabbit) startConsuming(queue, exchange, key string, maxRetries int, handler func(msg *amqp.Delivery) error) {
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
				if err := handler(&m); err != nil {
					// calculate retry count
					retryCount := 0
					if deaths, ok := m.Headers["x-death"].([]interface{}); ok && len(deaths) > 0 {
						if death, ok := deaths[0].(amqp.Table); ok {
							if count, ok := death["count"].(int64); ok {
								retryCount = int(count)
							}
						}
					}

					// if its any error other than "job already exists", we requeue
					shouldRequeue := retryCount < maxRetries && !errors.Is(err, k8s.ErrJobAlreadyExists)

					if shouldRequeue {
						log.Printf("Message failed, retrying (%d/%d): %v", retryCount+1, maxRetries, err)
					} else {
						log.Printf("Message failed, max retries reached or not retryable: %v", err)
					}

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
