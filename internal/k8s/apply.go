package k8s

import (
	"context"
	"d2c-gs-controller/internal/redis"
	"d2c-gs-controller/internal/util"
	"errors"

	"log"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/dota2classic/d2c-go-models/models"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

type DeployedMatch struct {
	ConfigMapName string
	SecretName    string
	JobName       string
}

var ErrJobAlreadyExists = errors.New("gameserver already running")

func DeployMatchResources(ctx context.Context, clientset *kubernetes.Clientset, evt *models.LaunchGameServerCommand) (*DeployedMatch, error) {

	password, err := util.GenerateSecureRandomString(12)

	if err != nil {
		password = "rconpassword"
	}

	gsPort, tvPort, err := redis.AllocateGameServerPorts()

	if err != nil {
		log.Printf("Error allocating game server ports: %v", err)
		return nil, err
	}

	runSchema, err := constructMatchInfoJson(evt)
	if err != nil {
		log.Printf("Error constructing MatchInfoJson: %v", err)
		return nil, err
	}

	priorityLobby := evt.LobbyType == models.MATCHMAKING_MODE_LOBBY || evt.LobbyType == models.MATCHMAKING_MODE_UNRANKED
	tickrate := 30
	cfgName := "server.cfg"
	if priorityLobby {
		tickrate = 40
		cfgName = "tickrate128.cfg"
	}

	data := templateData{
		MatchId:      evt.MatchID,
		GameMode:     evt.GameMode,
		LobbyType:    evt.LobbyType,
		Map:          evt.Map,
		Region:       evt.Region,
		RconPassword: password,
		MatchJson:    runSchema,
		TickRate:     tickrate,
		ConfigName:   cfgName,

		HostGamePort:     gsPort,
		HostSourceTVPort: tvPort,
	}

	// --- 1. CONFIGMAP ---
	configMap, err := ensureConfigMap(ctx, clientset, Namespace, &data)
	if err != nil {
		return nil, err
	}

	// --- 2. SECRET ---
	secret, err := ensureSecret(ctx, clientset, Namespace, &data)
	if err != nil {
		return nil, err
	}

	// --- 3. JOB ---
	job, err := createJob(ctx, clientset, Namespace, &data)
	if err != nil {
		return nil, err
	}

	return &DeployedMatch{
		ConfigMapName: configMap.Name,
		SecretName:    secret.Name,
		JobName:       job.Name,
	}, nil
}

func ensureConfigMap(ctx context.Context, clientset *kubernetes.Clientset, namespace string, data *templateData) (*corev1.ConfigMap, error) {
	configMap, err := createConfiguration[corev1.ConfigMap](ConfigmapTemplate, data)
	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Printf("ConfigMap already exists - updating")
			// Optionally you can update it instead:
			_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
			return configMap, nil
		}
		log.Printf("Error creating ConfigMap: %v", err)
		return nil, err
	}
	log.Println("Created ConfigMap")
	return configMap, nil
}

func ensureSecret(ctx context.Context, clientset *kubernetes.Clientset, namespace string, data *templateData) (*corev1.Secret, error) {
	secret, err := createConfiguration[corev1.Secret](SecretTemplate, data)
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Printf("Secret already exists - updating")
			// Optionally you can update it instead:
			_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
			return secret, nil
		}
		log.Printf("Error creating Secret: %v", err)
		return nil, err
	}
	log.Println("Created Secret")
	return secret, nil
}

func createJob(ctx context.Context, clientset *kubernetes.Clientset, namespace string, data *templateData) (*batchv1.Job, error) {
	job, err := createConfiguration[batchv1.Job](JobTemplate, data)
	_, err = clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})

	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Printf("Job already exists: we fail hard")
			return nil, ErrJobAlreadyExists
		}
		log.Printf("Error creating job: %v", err)
		return nil, err
	}
	log.Println("Created Job!")
	return job, nil
}
