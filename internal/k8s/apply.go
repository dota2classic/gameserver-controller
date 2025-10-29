package k8s

import (
	"context"
	"d2c-gs-controller/internal/util"

	"log"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/dota2classic/d2c-go-models/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DeployedMatch struct {
	ConfigMapName string
	SecretName    string
	JobName       string
}

func DeployMatchResources(ctx context.Context, clientset *kubernetes.Clientset, evt *models.LaunchGameServerCommand) (*DeployedMatch, error) {

	password, err := util.GenerateSecureRandomString(12)

	if err != nil {
		password = "rconpassword"
	}

	data := templateData{
		MatchId:      evt.MatchID,
		GameMode:     evt.GameMode,
		LobbyType:    evt.LobbyType,
		Map:          evt.Map,
		Region:       evt.Region,
		RconPassword: password,
	}

	namespace := "default"

	// --- 1. CONFIGMAP ---
	configMap, err := createConfiguration[corev1.ConfigMap](ConfigmapTemplate, &data)
	if err != nil {
		log.Printf("Error creating configmap %v", err)
		return nil, err
	}
	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Error creating configmap %v", err)
		return nil, err
	}
	log.Println("Created ConfigMap")

	// --- 2. SECRET ---
	secret, err := createConfiguration[corev1.Secret](SecretTemplate, &data)
	if err != nil {
		log.Printf("Error creating secret %v", err)
		return nil, err
	}
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Error creating secret %v", err)
		return nil, err
	}
	log.Println("Created Secret")

	// --- 3. JOB ---
	job, err := createConfiguration[batchv1.Job](JobTemplate, &data)
	if err != nil {
		log.Printf("Error creating job %v", err)
		return nil, err
	}
	_, err = clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Error creating job %v", err)
		return nil, err
	}
	log.Println("Created Job")

	return &DeployedMatch{
		ConfigMapName: configMap.Name,
		SecretName:    secret.Name,
		JobName:       job.Name,
	}, nil
}
