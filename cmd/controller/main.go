package main

import (
	"d2c-gs-controller/internal/monitoring"
	"d2c-gs-controller/internal/rabbit"
	"d2c-gs-controller/internal/redis"
	"log"
)

/**
What we need to do:
- Listen RMQ and launch game server on k8s
- Save jobs to database
- Update job statuses and emit events(ServerStatusEvent)
*/

func main() {
	rabbit.InitRabbitPublisher()
	redis.InitRedisClient()

	health := monitoring.NewHealthServer(redis.Client, rabbit.Client.Conn)
	err := health.Start(8080)
	if err != nil {
		log.Fatal(err)
	}
}
