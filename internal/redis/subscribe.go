package redis

import (
	"encoding/json"
	"log"
	"time"
)

type redisRequest[T any] struct {
	Id      string `json:"id"`
	Data    T      `json:"data"`
	Pattern string `json:"pattern"`
}

func Subscribe[In any, Out any](channel string, handler func(msg *In) (*Out, error)) {
	backoff := time.Second

	for {
		// Try to subscribe
		sub := Client.Subscribe(ctx, channel)
		ch := sub.Channel()

		log.Printf("[RedisSubscribe] Subscribed to %s", channel)

		// Listen until the subscription closes or context cancels
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					// Channel closed — Redis connection probably lost
					log.Printf("[RedisSubscribe] Channel closed for %s, will reconnect...", channel)
					goto reconnect
				}

				var event redisRequest[In]
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					log.Printf("[RedisSubscribe] Invalid message on %s: %v", channel, err)
					continue
				}

				res, err := handler(&event.Data)
				if err != nil {
					log.Printf("[RedisSubscribe] Handler error: %v", err)
					continue
				}

				if res == nil {
					continue // no reply
				}

				// you could publish reply here if needed
				// e.g. Client.Publish(ctx, event.ReplyTo, json.Marshal(res))

				replyChannel := channel + ".reply"

				response := redisRequest[Out]{
					Id:      event.Id,
					Data:    *res,
					Pattern: event.Pattern,
				}

				bt, err := json.Marshal(response)

				log.Printf("[RedisSubscribe] Publishing message to %s %v", channel, response)
				Client.Publish(ctx, replyChannel, bt)

			case <-ctx.Done():
				log.Printf("[RedisSubscribe] Context canceled for %s, exiting...", channel)
				_ = sub.Close()
				return
			}
		}

	reconnect:
		_ = sub.Close()
		log.Printf("[RedisSubscribe] Reconnecting to %s in %v...", channel, backoff)
		time.Sleep(backoff)

		// Exponential backoff up to 30s
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}
