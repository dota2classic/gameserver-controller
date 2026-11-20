package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dota2classic/d2c-go-models/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

const publishTimeout = 10 * time.Second

type publishFunc func(exchange, routingKey string, msg amqp.Publishing) error

func (r *Rabbit) PublishGameServerNotFound(matchID int64) error {
	return r.Publish("GameServerNotFoundEvent", &models.GameServerNotFoundEvent{MatchID: matchID})
}

func (r *Rabbit) Publish(routingKey string, event any) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return r.publishConfirmed(r.exchange, routingKey, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (r *Rabbit) publishConfirmed(exchange, routingKey string, msg amqp.Publishing) error {
	if r.Conn == nil || r.Conn.IsClosed() {
		return errors.New("rabbitmq connection is not open")
	}
	ch, err := r.Conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err := ch.Confirm(false); err != nil {
		return err
	}
	returns := ch.NotifyReturn(make(chan amqp.Return, 1))

	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()
	confirmation, err := ch.PublishWithDeferredConfirmWithContext(ctx, exchange, routingKey, true, false, msg)
	if err != nil {
		return err
	}
	acked, err := confirmation.WaitContext(ctx)
	if err != nil {
		return err
	}
	if !acked {
		return fmt.Errorf("broker rejected %s", routingKey)
	}
	select {
	case <-returns:
		return fmt.Errorf("no queue is bound to %s", routingKey)
	default:
		return nil
	}
}
