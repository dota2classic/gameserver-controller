package util

import (
	"log"
	"os"
	"time"
)

func GetEnvDuration(key, def string) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		val = def // default if env not set
	}
	var err error
	expiration, err := time.ParseDuration(val)
	if err != nil {
		log.Fatalf("Invalid %s: %v", key, err)
	}
	return expiration
}
