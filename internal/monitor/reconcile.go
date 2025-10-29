package monitor

import (
	"context"
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/k8s"
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
		}

		//Check Job status
	}

	return nil
}
