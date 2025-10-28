package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type HealthServer struct {
	redis  *redis.Client
	rabbit *amqp.Connection
}

func NewHealthServer(redis *redis.Client, rabbit *amqp.Connection) *HealthServer {
	return &HealthServer{
		redis:  redis,
		rabbit: rabbit,
	}
}

func (h *HealthServer) Liveness(w http.ResponseWriter, r *http.Request) {
	// Liveness just means: the process is alive.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *HealthServer) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	status := map[string]string{
		"redis":  "ok",
		"rabbit": "ok",
	}

	// Check Redis
	if err := h.redis.Ping(ctx).Err(); err != nil {
		status["redis"] = err.Error()
	}

	// Check RabbitMQ
	if h.rabbit == nil || h.rabbit.IsClosed() {
		status["rabbit"] = "not connected"
	}

	// Return combined result
	allHealthy := status["redis"] == "ok" && status["rabbit"] == "ok"

	w.Header().Set("Content-Type", "application/json")
	if allHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(status)
}

func (h *HealthServer) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.Liveness)
	mux.HandleFunc("/readyz", h.Readiness)
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}
