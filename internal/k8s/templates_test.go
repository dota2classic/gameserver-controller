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

func TestCreateCpuAffinityJob(t *testing.T) {
	job, err := createConfiguration[batchv1.Job](CpuAffinityJobTemplate, &data)
	jobName := fmt.Sprintf("gameserver-cpu-affinity-job-%d", data.MatchId)

	if err != nil {
		t.Errorf("Error creating job: %v", err)
	}

	if job.Name != jobName {
		t.Errorf("Resource name mismatch. Expected %s, got %s", jobName, job.Name)
	}

	// Should only run on gameserver nodes with selected region
	// Should prefer cpuAffinity nodes
	assertNodeAffinity(t, job.Spec.Template.Spec.Affinity,
		map[string][]string{
			"ru.dotaclassic/nodeType": {"gameserver"},
			"ru.dotaclassic/region":   {string(data.Region)},
		},
		map[string][]string{
			"dotaclassic.io/cpuAffinity": {"true"},
		})

	// Check sidecar envs
	sidecar := &job.Spec.Template.Spec.Containers[0]
	checkEnvVar(t, sidecar, "LOBBY_TYPE", strconv.FormatInt(int64(data.LobbyType), 10))
	checkEnvVar(t, sidecar, "GAME_MODE", strconv.FormatInt(int64(data.GameMode), 10))
	checkEnvVar(t, sidecar, "MATCH_ID", strconv.FormatInt(data.MatchId, 10))
	assertQosGuaranteed(t, sidecar, true)

	// Check GS envs
	gameserver := &job.Spec.Template.Spec.Containers[1]
	checkEnvVar(t, gameserver, "MAP", string(data.Map))
	checkEnvVar(t, gameserver, "GAMEMODE", strconv.FormatInt(int64(data.GameMode), 10))
	// We need cpu affinity and QoS = guaranteed here
	assertQosGuaranteed(t, gameserver, true)
	assertCpuAffinity(t, gameserver, true)
}

func TestCreateRegularJob(t *testing.T) {
	job, err := createConfiguration[batchv1.Job](RegularJobTemplate, &data)
	jobName := fmt.Sprintf("gameserver-regular-job-%d", data.MatchId)

	if err != nil {
		t.Errorf("Error creating job: %v", err)
	}

	if job.Name != jobName {
		t.Errorf("Resource name mismatch. Expected %s, got %s", jobName, job.Name)
	}

	// Should only run on gameserver nodes with selected region
	// Should prefer non-cpuAffinity nodes
	assertNodeAffinity(t, job.Spec.Template.Spec.Affinity,
		map[string][]string{
			"ru.dotaclassic/nodeType": {"gameserver"},
			"ru.dotaclassic/region":   {string(data.Region)},
		},
		map[string][]string{
			"dotaclassic.io/cpuAffinity": {"false"},
		})

	// Check sidecar envs
	sidecar := &job.Spec.Template.Spec.Containers[0]
	checkEnvVar(t, sidecar, "LOBBY_TYPE", strconv.FormatInt(int64(data.LobbyType), 10))
	checkEnvVar(t, sidecar, "GAME_MODE", strconv.FormatInt(int64(data.GameMode), 10))
	checkEnvVar(t, sidecar, "MATCH_ID", strconv.FormatInt(data.MatchId, 10))
	assertQosGuaranteed(t, sidecar, false)

	// Check GS envs
	gameserver := &job.Spec.Template.Spec.Containers[1]
	checkEnvVar(t, gameserver, "MAP", string(data.Map))
	checkEnvVar(t, gameserver, "GAMEMODE", strconv.FormatInt(int64(data.GameMode), 10))
	// We don't need cpu affinity and QoS = guaranteed here
	assertQosGuaranteed(t, gameserver, false)
	assertCpuAffinity(t, gameserver, false)
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

func assertQosGuaranteed(t *testing.T, container *corev1.Container, expectTrue bool) {
	cpuEqual := container.Resources.Requests.Cpu().MilliValue() == container.Resources.Limits.Cpu().MilliValue()
	memEqual := container.Resources.Requests.Memory().MilliValue() == container.Resources.Limits.Memory().MilliValue()
	isGuaranteed := cpuEqual && memEqual

	if expectTrue && !isGuaranteed {
		t.Errorf("Expected container to be QoS=Guaranteed, but cpu or memory request != limit")
	}

	if !expectTrue && isGuaranteed {
		t.Errorf("Expected container NOT to be QoS=Guaranteed, but cpu and memory requests equal limits")
	}
}

func assertCpuAffinity(t *testing.T, container *corev1.Container, expectTrue bool) {
	cpuMilli := container.Resources.Requests.Cpu().MilliValue()
	hasAffinity := cpuMilli%1000 == 0

	if expectTrue && !hasAffinity {
		t.Errorf("Expected container to have CPU affinity, but cpu request is non-integer: %dm", cpuMilli)
	}

	if !expectTrue && hasAffinity {
		t.Errorf("Expected container NOT to have CPU affinity, but cpu request is integer: %dm", cpuMilli)
	}
}

func assertNodeAffinity(t *testing.T, affinity *corev1.Affinity, expectedRequired map[string][]string, expectedPreferred map[string][]string) {
	t.Helper()

	if affinity == nil || affinity.NodeAffinity == nil {
		t.Fatalf("NodeAffinity is missing")
	}

	// --- Required checks ---
	required := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if required == nil {
		t.Fatalf("RequiredDuringSchedulingIgnoredDuringExecution is missing")
	}

	// Track which required keys we saw
	seenRequired := map[string]bool{}
	for _, term := range required.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			if wantVals, ok := expectedRequired[expr.Key]; ok {
				if expr.Operator != corev1.NodeSelectorOpIn {
					t.Errorf("required key %s: operator = %s, want In", expr.Key, expr.Operator)
				}
				for _, val := range wantVals {
					if !contains(expr.Values, val) {
						t.Errorf("required key %s: missing value %s", expr.Key, val)
					}
				}
				seenRequired[expr.Key] = true
			}
		}
	}
	for key := range expectedRequired {
		if !seenRequired[key] {
			t.Errorf("required key %s missing in NodeAffinity", key)
		}
	}

	// --- Preferred checks ---
	preferred := affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	for _, pref := range preferred {
		for _, expr := range pref.Preference.MatchExpressions {
			if wantVals, ok := expectedPreferred[expr.Key]; ok {
				if expr.Operator != corev1.NodeSelectorOpIn {
					t.Errorf("preferred key %s: operator = %s, want In", expr.Key, expr.Operator)
				}
				for _, val := range wantVals {
					if !contains(expr.Values, val) {
						t.Errorf("preferred key %s: missing value %s", expr.Key, val)
					}
				}
			}
		}
	}
}

// helper for checking slice
func contains(slice []string, val string) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
