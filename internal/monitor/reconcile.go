package monitor

import (
	"context"
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/k8s"
	"d2c-gs-controller/internal/redis"
	"d2c-gs-controller/internal/util"
	"encoding/json"
	"log"
	"time"

	//"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func reconcileMatches() error {
	matchResources, err := db.FindAllMatchResources()

	if err != nil {
		log.Printf("failed to find matchResources in db: %v", err)
	}

	client := k8s.GetClient()

	for _, mr := range matchResources {
		// Query job from Kubernetes
		job, err := client.BatchV1().Jobs(k8s.Namespace).Get(context.Background(), mr.JobName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Job does not exist anymore → cleanup
				deleteJobAndResources(client, &mr)
				continue
			}
			log.Printf("Failed to get job %s: %v", mr.JobName, err)
			continue
		}

		jobStatus := getJobStatus(context.Background(), client, job)

		err = db.UpdateStatus(mr.MatchId, jobStatus)
		if err != nil {
			log.Printf("failed to update status for job %s: %v", mr.JobName, err)
		}

		switch jobStatus {
		case db.StatusLaunching:
			if mr.CreatedAt.Add(getExpirationTimeout()).Before(time.Now()) {
				log.Printf("Cancelling stale job %s", mr.JobName)
				deleteJobAndResources(client, &mr)
				emitNoFreeServer(&mr)
			}
		case db.StatusDone:
			log.Printf("Job %s done, cleaning up resources", mr.JobName)
			deleteJobAndResources(client, &mr)
		case db.StatusRunning:
			log.Printf("Job %s is running", mr.JobName)

			if err != nil {
				log.Printf("Failed to publish status event for job %s: %v", mr.JobName, err)
			}
		}

		// Check Job status
	}

	return nil
}

func checkHeartbeats() error {
	ctx := context.Background()
	client := redis.Client

	keys, err := client.Keys(ctx, "server:*").Result()
	if err != nil {
		return err
	}

	now := time.Now()
	timeout := 40 * time.Second

	for _, key := range keys {
		raw, err := client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var info util.ServerInfo
		if err := json.Unmarshal([]byte(raw), &info); err != nil {
			continue
		}

		ts := time.Unix(info.Timestamp, 0)

		if now.Sub(ts) > timeout {
			// Server considered dead
			redis.ServerStatus(info.URL, false)

			// Optional: delete the key
			client.Del(ctx, key)
		} else {
			// Server alive
			redis.ServerStatus(info.URL, true)
		}
	}

	return nil
}
