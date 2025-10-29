package monitor

import (
	"context"
	"d2c-gs-controller/internal/db"
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
	// 1. First check if it's done
	if job.Status.Succeeded > 0 {
		return db.StatusDone
	}
	if job.Status.Failed > 0 {
		return db.StatusFailed
	}

	// 2. Then inspect pods for readiness
	pods, err := client.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
	})
	if err != nil {
		log.Printf("Failed to list pods for job %s: %v", job.Name, err)
		return db.StatusLaunching // safest fallback
	}

	if len(pods.Items) == 0 {
		return db.StatusLaunching // Job created, pods not yet started
	}

	allRunningAndReady := true
	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodPending, corev1.PodUnknown:
			return db.StatusLaunching
		case corev1.PodRunning:
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					allRunningAndReady = false
					break
				}
			}
		case corev1.PodFailed:
			return db.StatusFailed
		case corev1.PodSucceeded:
			return db.StatusDone
		}
	}

	if allRunningAndReady {
		return db.StatusRunning
	}

	return db.StatusLaunching
}

func getStatusFromJob(job *batchv1.Job) db.Status {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return db.StatusDone
		}
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return db.StatusFailed
		}
	}
	return db.StatusRunning
}

func deleteJobAndResources(client *kubernetes.Clientset, mr *db.MatchResources) {
	ctx := context.Background()

	deletePolicy := metav1.DeletePropagationBackground

	// delete Job
	_ = client.BatchV1().Jobs("default").Delete(ctx, mr.JobName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	// delete ConfigMap
	_ = client.CoreV1().ConfigMaps("default").Delete(ctx, mr.ConfigMapName, metav1.DeleteOptions{})

	// delete Secret
	_ = client.CoreV1().Secrets("default").Delete(ctx, mr.SecretName, metav1.DeleteOptions{})

	// delete DB row
	db.DeleteMatchResources(mr.MatchId)
}

func emitNoFreeServer(mr *db.MatchResources) {

}

func getExpirationTimeout() time.Duration {
	return util.GetEnvDuration("GAMESERVER_EXPIRATION_TIMEOUT", "2m")
}
