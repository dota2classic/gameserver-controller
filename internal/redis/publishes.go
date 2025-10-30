package redis

import (
	"log"

	"github.com/dota2classic/d2c-go-models/models"
)

func ServerStatus(url string, alive bool) {
	err := publishWithRetry("ServerStatusEvent", &models.ServerStatusEvent{
		Url:       url,
		IsRunning: alive,
	}, 1)
	if err != nil {
		log.Printf("There was an issue publishing event: %v\n", err)
	}
}
