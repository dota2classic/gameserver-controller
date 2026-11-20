package rabbit

import (
	"errors"
	"testing"
	"time"

	"github.com/dota2classic/d2c-go-models/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

type fakeBroker struct {
	acks, nacks int
	retryQueue  []amqp.Publishing
}

func (b *fakeBroker) Ack(uint64, bool) error        { b.acks++; return nil }
func (b *fakeBroker) Nack(uint64, bool, bool) error { b.nacks++; return nil }
func (b *fakeBroker) Reject(uint64, bool) error     { b.nacks++; return nil }

func (b *fakeBroker) publish(t *testing.T) publishFunc {
	return func(exchange, routingKey string, msg amqp.Publishing) error {
		if exchange != "" || routingKey != "work.retry" {
			t.Fatalf("retry published to %q/%q, want the default exchange and work.retry", exchange, routingKey)
		}
		if msg.Expiration != "5000" {
			t.Fatalf("retry expiration = %q, want 5000", msg.Expiration)
		}
		if _, ok := msg.Headers["x-death"]; ok {
			t.Fatal("broker-owned x-death header was republished")
		}
		b.retryQueue = append(b.retryQueue, msg)
		return nil
	}
}

func (b *fakeBroker) deadLetterBack() *amqp.Delivery {
	if len(b.retryQueue) == 0 {
		return nil
	}
	msg := b.retryQueue[0]
	b.retryQueue = b.retryQueue[1:]
	headers := amqp.Table{"x-death": []interface{}{amqp.Table{"count": int64(1)}}}
	for k, v := range msg.Headers {
		headers[k] = v
	}
	return &amqp.Delivery{Acknowledger: b, Headers: headers, Body: msg.Body}
}

func TestFailingLaunchIsRetriedWithDelayThenReportedAsNotFound(t *testing.T) {
	failures := map[string]func(){
		"error": func() {},
		"panic": func() { panic("nil pointer dereference") },
	}
	for name, fail := range failures {
		t.Run(name, func(t *testing.T) {
			const maxRetries = 3
			attempts, giveUps := 0, 0
			var reported []int64
			c := consumer[models.LaunchGameServerCommand]{
				queue: "work",
				retry: retryPolicy{delay: 5 * time.Second, maxRetries: maxRetries},
				handle: func(*models.LaunchGameServerCommand) error {
					attempts++
					fail()
					return errors.New("OOM command not allowed when used memory > 'maxmemory'")
				},
				retryable: isRetryableLaunchError,
				giveUp: func(cmd *models.LaunchGameServerCommand) error {
					giveUps++
					if giveUps == 1 {
						return errors.New("broker unavailable")
					}
					reported = append(reported, cmd.MatchID)
					return nil
				},
			}

			broker := &fakeBroker{}
			deliveries := 0
			for d := (&amqp.Delivery{Acknowledger: broker, Body: []byte(`{"matchId":42}`)}); d != nil; d = broker.deadLetterBack() {
				deliveries++
				if deliveries > 3*maxRetries {
					t.Fatal("retry loop did not terminate")
				}
				c.process(d, broker.publish(t))
			}

			if attempts != maxRetries+1 {
				t.Errorf("launch attempts = %d, want %d", attempts, maxRetries+1)
			}
			if len(reported) != 1 || reported[0] != 42 {
				t.Errorf("GameServerNotFoundEvent published for %v, want [42]", reported)
			}
			if broker.acks != deliveries || broker.nacks != 0 {
				t.Errorf("acks = %d, nacks = %d over %d deliveries; want every delivery acked once", broker.acks, broker.nacks, deliveries)
			}
		})
	}
}
