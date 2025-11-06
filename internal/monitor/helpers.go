package monitor

import (
	"context"
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/k8s"
	"d2c-gs-controller/internal/util"
	"fmt"
	"log"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func isPodFullyRunning(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	// All containers must be ready
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			return false
		}
	}

	return true
}

func getJobStatus(ctx context.Context, client *kubernetes.Clientset, job *batchv1.Job) db.Status {
	// 1. Check job-level completion first
	if job.Status.Succeeded > 0 {
		return db.StatusDone
	}
	if job.Status.Failed > 0 {
		return db.StatusFailed
	}

	// 2. Check pods associated with this job
	pods, err := client.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
	})
	if err != nil {
		log.Printf("Failed to list pods for job %s: %v", job.Name, err)
		return db.StatusLaunching
	}
	if len(pods.Items) == 0 {
		return db.StatusPending
	}

	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodPending:
			// Not scheduled yet
			if pod.Spec.NodeName == "" {
				return db.StatusPending
			}
			// Scheduled but containers not yet launched
			return db.StatusLaunching

		case corev1.PodRunning:
			// Inspect containers
			sidecarAlive := false
			mainAlive := false
			allReady := true

			for _, cs := range pod.Status.ContainerStatuses {
				if cs.Name == "sidecar" && cs.Ready {
					sidecarAlive = true
				}
				if cs.Name != "sidecar" && cs.Ready {
					mainAlive = true
				}
				if !cs.Ready {
					allReady = false
				}
			}

			// Finishing: only sidecar alive
			if sidecarAlive && !mainAlive {
				return db.StatusFinishing
			}

			// Running: both alive and ready
			if sidecarAlive && mainAlive && allReady {
				return db.StatusRunning
			}

			// Otherwise, launching (some containers not ready yet)
			return db.StatusLaunching

		case corev1.PodSucceeded:
			return db.StatusDone

		case corev1.PodFailed:
			return db.StatusFailed
		}
	}

	// Default fallback
	return db.StatusLaunching
}

func deleteJobAndResources(client *kubernetes.Clientset, mr *db.MatchResources) {
	ctx := context.Background()

	deletePolicy := metav1.DeletePropagationBackground

	// delete Job
	_ = client.BatchV1().Jobs(k8s.Namespace).Delete(ctx, mr.JobName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	// delete ConfigMap
	_ = client.CoreV1().ConfigMaps(k8s.Namespace).Delete(ctx, mr.ConfigMapName, metav1.DeleteOptions{})

	// delete Secret
	_ = client.CoreV1().Secrets(k8s.Namespace).Delete(ctx, mr.SecretName, metav1.DeleteOptions{})

	// delete DB row
	db.DeleteMatchResources(mr.MatchId)
}

func emitNoFreeServer(mr *db.MatchResources) {

}

func getExpirationTimeout() time.Duration {
	return util.GetEnvDuration("GAMESERVER_EXPIRATION_TIMEOUT", "2m")
}
