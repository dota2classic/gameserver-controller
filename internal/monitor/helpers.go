package monitor

import (
	"context"
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/util"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func isJobPending(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed {
			return false
		}
	}
	return true
}

func isJobDone(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
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
