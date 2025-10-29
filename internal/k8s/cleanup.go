package k8s

import (
	"context"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CleanupDeployedMatch(ctx context.Context, clientset *kubernetes.Clientset, deployment *DeployedMatch) {
	namespace := "default"
	err := clientset.BatchV1().Jobs(namespace).Delete(ctx, deployment.JobName, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("Error deleting job: %v", err)
	}
	err = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, deployment.ConfigMapName, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("Error deleting configmap: %v", err)
	}
	err = clientset.CoreV1().Secrets(namespace).Delete(ctx, deployment.SecretName, metav1.DeleteOptions{})
	if err != nil {
		log.Printf("Error deleting secret: %v", err)
	}
}
