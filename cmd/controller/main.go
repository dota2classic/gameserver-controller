package main

import (
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/monitor"
	"d2c-gs-controller/internal/monitoring"
	"d2c-gs-controller/internal/rabbit"
	"d2c-gs-controller/internal/redis"
	"log"

	"github.com/joho/godotenv"
)

/**
What we need to do:
- Listen RMQ and launch game server on k8s
- Save jobs to database
- Update job statuses and emit events(ServerStatusEvent)
*/

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	db.ConnectAndMigrate()
	rabbit.InitRabbit()
	redis.InitRedisClient()

	go monitor.CronMatchResourceStatus()

	health := monitoring.NewHealthServer(redis.Client, rabbit.Instance.Conn)
	log.Println("Starting server")
	if health.Start(8080) != nil {
		log.Fatal(err)
	}
}
