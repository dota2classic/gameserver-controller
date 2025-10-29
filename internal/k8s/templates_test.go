package k8s

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/dota2classic/d2c-go-models/models"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

var data = templateData{
	MatchId:      123,
	GameMode:     models.DOTA_GAME_MODE_ALLPICK,
	LobbyType:    models.MATCHMAKING_MODE_LOBBY,
	Map:          models.DOTA_MAP_DOTA,
	Region:       models.REGION_RU_MOSCOW,
	RconPassword: "12345",
	MatchJson:    `{"key":"value"}`,
}

func TestCreateConfigmap(t *testing.T) {
	cfg, err := createConfiguration[corev1.ConfigMap](ConfigmapTemplate, &data)
	configName := fmt.Sprintf("gameserver-config-%d", data.MatchId)

	if err != nil {
		t.Errorf("Error creating configmap: %v", err)
	}

	if !jsonEqual(cfg.Data["match.json"], data.MatchJson) {
		t.Errorf("Configmap: match.json does not match the expected json. Expected %s, got %s", data.MatchJson, cfg.Data["match.json"])
	}

	if cfg.ObjectMeta.Name != configName {
		t.Errorf("Resource name mismatch. Expected %s, got %s", configName, cfg.Name)
	}
}

func TestCreateSecret(t *testing.T) {
	cfg, err := createConfiguration[corev1.Secret](SecretTemplate, &data)
	secretName := fmt.Sprintf("gameserver-secrets-%d", data.MatchId)

	if err != nil {
		t.Errorf("Error creating secret: %v", err)
	}

	if !jsonEqual(cfg.StringData["RCON_PASSWORD"], data.RconPassword) {
		t.Errorf("Secret: rcon password mismatch. Expected %s, got %s", data.RconPassword, cfg.StringData["RCON_PASSWORD"])
	}

	if cfg.Name != secretName {
		t.Errorf("Resource name mismatch. Expected %s, got %s", secretName, cfg.Name)
	}
}

func TestCreateJob(t *testing.T) {
	job, err := createConfiguration[batchv1.Job](JobTemplate, &data)
	jobName := fmt.Sprintf("gameserver-job-%d", data.MatchId)

	if err != nil {
		t.Errorf("Error creating job: %v", err)
	}

	if job.Name != jobName {
		t.Errorf("Resource name mismatch. Expected %s, got %s", jobName, job.Name)
	}

	if job.Spec.Template.Spec.NodeSelector["region"] != string(data.Region) {
		t.Errorf("Region mismatch")
	}

	if job.Spec.Template.Spec.NodeSelector["gs-node"] != "true" {
		t.Errorf("gs-node selector != true")
	}

	sidecar := &job.Spec.Template.Spec.Containers[0]

	checkEnvVar(t, sidecar, "LOBBY_TYPE", strconv.FormatInt(int64(data.LobbyType), 10))
	checkEnvVar(t, sidecar, "GAME_MODE", strconv.FormatInt(int64(data.GameMode), 10))
	checkEnvVar(t, sidecar, "MATCH_ID", strconv.FormatInt(data.MatchId, 10))

	gameserver := &job.Spec.Template.Spec.Containers[1]
	checkEnvVar(t, gameserver, "MAP", string(data.Map))
	checkEnvVar(t, gameserver, "GAMEMODE", strconv.FormatInt(int64(data.GameMode), 10))

}

func jsonEqual(a, b string) bool {
	var o1 interface{}
	var o2 interface{}

	if err := json.Unmarshal([]byte(a), &o1); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &o2); err != nil {
		return false
	}

	return reflect.DeepEqual(o1, o2)
}

func checkEnvVar(t *testing.T, container *corev1.Container, name string, value string) {
	for _, env := range container.Env {
		if env.Name == name {
			if env.Value != value {
				t.Errorf("Container env var %s value mismatch. Expected %s, got %s", name, value, env.Value)
			}
			return
		}
	}
	t.Errorf("Env var %s not found in container %s", name, container.Name)
}
