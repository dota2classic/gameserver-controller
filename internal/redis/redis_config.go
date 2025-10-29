package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dota2classic/d2c-go-models/util"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var Client *redis.Client

func InitRedisClient() {
	host := os.Getenv("REDIS_HOST")
	port := util.GetEnvInt("REDIS_PORT", 6379)

	password := os.Getenv("REDIS_PASSWORD")

	// Create Client
	Client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           0,
		PoolSize:     2,
		MinIdleConns: 1,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	if err := Client.Ping(ctx).Err(); err != nil {
		log.Fatal(err)
	}

	log.Println("Redis Client initialized")
}

// publishWithRetry publishes a message with automatic retry logic.
func publishWithRetry[T any](channel string, event *T, retries int) error {
	var err error

	message, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("can't serialize payload: %w", err)
	}

	for attempt := 1; attempt <= retries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = Client.Publish(ctx, channel, message).Err()
		cancel()

		if err == nil {
			return nil
		}

		// Check if it's a transient error (network, closed conn, etc.)
		if errors.Is(err, redis.ErrClosed) || errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("⚠️  Redis publish failed (attempt %d/%d): %v — retrying...\n", attempt, retries, err)
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
			continue
		}

		// Non-retryable error
		return fmt.Errorf("redis publish failed: %w", err)
	}

	return fmt.Errorf("redis publish failed after %d retries: %w", retries, err)
}
