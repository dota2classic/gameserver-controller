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

		r.startConsuming(fmt.Sprintf("d2c-gs-controller.LaunchGameServerCommand.%s", region), Exchange, key, func(msg []byte) error {
			var event models.LaunchGameServerCommand
			if err := json.Unmarshal(msg, &event); err != nil {
				return err
			}
			return queues.HandleLaunchGameServerCommand(&event)
		})

		r.startConsuming("d2c-gs-controller.KillServerRequestedEvent", Exchange, "KillServerRequestedEvent", func(msg []byte) error {
			var event models.KillServerRequestedEvent
			if err := json.Unmarshal(msg, &event); err != nil {
				return err
			}
			return queues.HandleKillServer(&event)
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
