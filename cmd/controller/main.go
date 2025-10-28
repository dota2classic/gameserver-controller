package main

import (
	"d2c-gs-controller/internal/rabbit"
	"d2c-gs-controller/internal/redis"
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
}
