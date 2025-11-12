package main

import (
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/monitor"
	"d2c-gs-controller/internal/monitoring"
	"d2c-gs-controller/internal/rabbit"
	"d2c-gs-controller/internal/redis"
	"log"

	"github.com/dota2classic/d2c-go-models/models"
	"github.com/joho/godotenv"
)

/**
What we need to do:
- Listen RMQ and launch game server on k8s
- Save jobs to database
- Update job statuses and emit events(ServerStatusEvent)
*/

type void struct{}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	db.ConnectAndMigrate()
	rabbit.InitRabbit()
	redis.InitRedisClient()

	go redis.Subscribe("KillServerRequestedEvent", func(msg *models.KillServerRequestedEvent) (*void, error) {
		return nil, monitor.KillServer(msg.MatchID)
	})

	go monitor.CronMatchResourceStatus()
	go monitor.CronServerHeartbeats()

	health := monitoring.NewHealthServer(redis.Client, rabbit.Instance.Conn)
	log.Println("Starting server")
	if health.Start(8080) != nil {
		log.Fatal(err)
	}
}
