package rabbit

import (
	"d2c-gs-controller/internal/k8s"
	"d2c-gs-controller/internal/rabbit/queues"
	"d2c-gs-controller/internal/util"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/dota2classic/d2c-go-models/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

const retryCountHeader = "x-retry-count"

var regions = []models.Region{
	models.REGION_RU_MOSCOW,
	models.REGION_RU_NOVOSIBIRSK,
	models.REGION_EU_CZECH,
}

type retryPolicy struct {
	delay      time.Duration
	maxRetries int
}

type consumer[T any] struct {
	queue     string
	key       string
	retry     retryPolicy
	handle    func(*T) error
	retryable func(error) bool
	giveUp    func(*T) error
}

func launchRetryPolicy() retryPolicy {
	delay := util.GetEnvDuration("GAMESERVER_LAUNCH_RETRY_DELAY", "5s")
	timeout := util.GetEnvDuration("GAMESERVER_LAUNCH_RETRY_TIMEOUT", "2m")
	if delay < time.Millisecond {
		log.Fatalf("GAMESERVER_LAUNCH_RETRY_DELAY must be at least 1ms, got %v", delay)
	}
	if timeout < 0 {
		log.Fatalf("GAMESERVER_LAUNCH_RETRY_TIMEOUT must not be negative, got %v", timeout)
	}
	return retryPolicy{delay: delay, maxRetries: int(timeout / delay)}
}

func (r *Rabbit) initConsumers() {
	retry := launchRetryPolicy()
	for _, region := range regions {
		c := consumer[models.LaunchGameServerCommand]{
			queue:  fmt.Sprintf("d2c-gs-controller.LaunchGameServerCommand.%s", region),
			key:    fmt.Sprintf("LaunchGameServerCommand.%s", region),
			retry:  retry,
			handle: queues.HandleLaunchGameServerCommand,
			retryable: func(err error) bool {
				return !errors.Is(err, k8s.ErrJobAlreadyExists)
			},
			giveUp: func(cmd *models.LaunchGameServerCommand) error {
				return r.PublishGameServerNotFound(cmd.MatchID)
			},
		}
		go c.run(r)
	}
}

func (c consumer[T]) retryQueue() string {
	return c.queue + ".retry"
}

func (c consumer[T]) run(r *Rabbit) {
	for {
		ch, err := r.getChannel()
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		msgs, err := c.subscribe(ch)
		if err != nil {
			log.Printf("[%s] Subscribe failed: %v", c.queue, err)
			ch.Close()
			time.Sleep(2 * time.Second)
			continue
		}

		for m := range msgs {
			c.process(&m, r.publishConfirmed)
		}

		ch.Close()
		time.Sleep(2 * time.Second)
	}
}

func (c consumer[T]) subscribe(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	if _, err := ch.QueueDeclare(c.queue, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("declare %s: %w", c.queue, err)
	}
	if err := ch.QueueBind(c.queue, c.key, Exchange, false, nil); err != nil {
		return nil, fmt.Errorf("bind %s: %w", c.queue, err)
	}
	if _, err := ch.QueueDeclare(c.retryQueue(), true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": c.queue,
	}); err != nil {
		return nil, fmt.Errorf("declare %s: %w", c.retryQueue(), err)
	}
	return ch.Consume(c.queue, "", false, false, false, false, nil)
}

func (c consumer[T]) process(m *amqp.Delivery, publish publishFunc) {
	var msg T
	if err := json.Unmarshal(m.Body, &msg); err != nil {
		log.Printf("[%s] Dropping malformed message: %v", c.queue, err)
		m.Nack(false, false)
		return
	}

	failures := failureCount(m)
	if failures <= c.retry.maxRetries {
		err := c.handle(&msg)
		if err == nil {
			m.Ack(false)
			return
		}
		if !c.retryable(err) {
			log.Printf("[%s] Dropping message, error is not retryable: %v", c.queue, err)
			m.Nack(false, false)
			return
		}
		failures++
		log.Printf("[%s] Attempt %d/%d failed: %v", c.queue, failures, c.retry.maxRetries+1, err)
	}

	if failures > c.retry.maxRetries {
		err := c.giveUp(&msg)
		if err == nil {
			log.Printf("[%s] Gave up after %d failed attempts", c.queue, failures)
			m.Ack(false)
			return
		}
		log.Printf("[%s] Give-up action failed, retrying it in %v: %v", c.queue, c.retry.delay, err)
	}

	if err := publish("", c.retryQueue(), retryPublishing(m, failures, c.retry.delay)); err != nil {
		log.Printf("[%s] Failed to schedule retry, requeueing in %v: %v", c.queue, c.retry.delay, err)
		time.Sleep(c.retry.delay)
		m.Nack(false, true)
		return
	}
	m.Ack(false)
}

func failureCount(m *amqp.Delivery) int {
	switch n := m.Headers[retryCountHeader].(type) {
	case int32:
		return int(n)
	case int64:
		return int(n)
	}
	return 0
}

func retryPublishing(m *amqp.Delivery, failures int, delay time.Duration) amqp.Publishing {
	headers := amqp.Table{}
	for k, v := range m.Headers {
		if !isDeathHeader(k) {
			headers[k] = v
		}
	}
	headers[retryCountHeader] = int32(failures)
	return amqp.Publishing{
		Headers:         headers,
		ContentType:     m.ContentType,
		ContentEncoding: m.ContentEncoding,
		DeliveryMode:    amqp.Persistent,
		CorrelationId:   m.CorrelationId,
		MessageId:       m.MessageId,
		Timestamp:       m.Timestamp,
		Type:            m.Type,
		AppId:           m.AppId,
		Expiration:      strconv.FormatInt(delay.Milliseconds(), 10),
		Body:            m.Body,
	}
}

func isDeathHeader(key string) bool {
	return key == "x-death" || strings.HasPrefix(key, "x-first-death-") || strings.HasPrefix(key, "x-last-death-")
}
