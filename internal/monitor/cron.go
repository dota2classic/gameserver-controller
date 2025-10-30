package monitor

import (
	"d2c-gs-controller/internal/util"
	"log"
	"time"
)

func CronMatchResourceStatus() {
	interval := util.GetEnvDuration("POD_CHECK_INTERVAL", "30s")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := reconcileMatches()
			if err != nil {
				log.Printf("Reconcile error: %v", err)
			}
		}
	}
}

func CronServerHeartbeats() {
	interval := time.Second * 5
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := checkHeartbeats()
			if err != nil {
				log.Printf("Check heartbeats error: %v", err)
			}
		}
	}
}
