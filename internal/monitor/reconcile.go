package monitor

import (
	"context"
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/k8s"
	"log"
	"time"

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
		job, err := client.BatchV1().Jobs("default").Get(context.Background(), mr.JobName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Job does not exist anymore → cleanup
				deleteJobAndResources(client, &mr)
				continue
			}
			log.Printf("Failed to get job %s: %v", mr.JobName, err)
			continue
		}

		// Check Job status
		switch {
		case isJobPending(job) && mr.CreatedAt.Add(getExpirationTimeout()).Before(time.Now()):
			log.Printf("Cancelling stale job %s", mr.JobName)
			deleteJobAndResources(client, &mr)
			emitNoFreeServer(&mr)

		case isJobDone(job):
			log.Printf("Job %s done, cleaning up resources", mr.JobName)
			deleteJobAndResources(client, &mr)

		default:
			// update db status
			status := getStatusFromJob(job)
			if status != mr.Status {
				err = db.UpdateStatus(mr.MatchId, status)
				if err != nil {
					log.Printf("Failed to update status for %d: %v", mr.MatchId, err)
				}
			}
		}
	}

	return nil
}
