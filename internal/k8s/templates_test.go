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
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
)

var sidecarResources = corev1.ResourceRequirements{
	Requests: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("25m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	},
	Limits: corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("256Mi"),
	},
}

const gamePriorityClass = "game-server"

var gameserverCommand = []string{
	"perl",
	"-e",
	`my $attr=pack("LLQlLQQQ",48,0,0,0,0,500000,0,0);syscall(314,0,$attr,0)==0 or print STDERR "sched_setattr failed: $!\n";exec {$ARGV[0]} @ARGV or die "exec failed: $!\n";`,
	"/root/entrypoint.sh",
}

var gameserverResources = corev1.ResourceRequirements{
	Requests: corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("425m"),
		corev1.ResourceMemory: resource.MustParse("512Mi"),
	},
	Limits: corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("1Gi"),
	},
}

var data = templateData{
	MatchId:         123,
	GameMode:        models.DOTA_GAME_MODE_ALLPICK,
	LobbyType:       models.MATCHMAKING_MODE_LOBBY,
	Map:             models.DOTA_MAP_DOTA,
	Region:          models.REGION_RU_MOSCOW,
	RconPassword:    "12345",
	MatchJson:       `{"key":"value"}`,
	GameServerImage: "myimage123",
}

func TestCPUReservationByModeInBothTemplates(t *testing.T) {
	cases := []struct {
		name string
		mode models.MatchmakingMode
		game models.DotaGameMode
		cpu  string
	}{
		{"allpick", models.MATCHMAKING_MODE_UNRANKED, models.DOTA_GAME_MODE_ALLPICK, "600m"},
		{"rankedAllPickUsedByPublicQueue", models.MATCHMAKING_MODE_UNRANKED, models.DOTA_GAME_MODE_RANKED_AP, "600m"},
		{"bots", models.MATCHMAKING_MODE_BOTS, models.DOTA_GAME_MODE_ALLPICK, "300m"},
		{"bots2x2", models.MATCHMAKING_MODE_BOTS_2X2, models.DOTA_GAME_MODE_ALLPICK, "300m"},
		{"solomid", models.MATCHMAKING_MODE_SOLOMID, models.DOTA_GAME_MODE_ALLPICK, "250m"},
		{"tournamentSolomid", models.MATCHMAKING_MODE_TOURNAMENT_SOLOMID, models.DOTA_GAME_MODE_ALLPICK, "250m"},
		{"privateLobbyUnchanged", models.MATCHMAKING_MODE_LOBBY, models.DOTA_GAME_MODE_ALLPICK, "425m"},
		{"highroomUnchanged", models.MATCHMAKING_MODE_HIGHROOM, models.DOTA_GAME_MODE_ALLPICK, "425m"},
		{"otherGameModeUnchanged", models.MATCHMAKING_MODE_UNRANKED, models.DotaGameMode(2), "425m"},
	}
	for name, tmpl := range map[string]string{"regular": RegularJobTemplate, "affinity": CpuAffinityJobTemplate} {
		for _, tc := range cases {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				input := data
				input.LobbyType, input.GameMode = tc.mode, tc.game
				job, err := createConfiguration[batchv1.Job](tmpl, &input)
				if err != nil {
					t.Fatal(err)
				}
				game := job.Spec.Template.Spec.Containers[1]
				expected := gameserverResources.DeepCopy()
				expected.Requests[corev1.ResourceCPU] = resource.MustParse(tc.cpu)
				if !apiequality.Semantic.DeepEqual(game.Resources, *expected) {
					t.Fatalf("unexpected game resources: %v; want %v", game.Resources, *expected)
				}
				if !apiequality.Semantic.DeepEqual(job.Spec.Template.Spec.Containers[0].Resources, sidecarResources) {
					t.Fatal("sidecar resources changed")
				}
				if _, capped := game.Resources.Limits[corev1.ResourceCPU]; capped {
					t.Fatal("CPU limit must remain absent")
				}
			})
		}
	}
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
	//assertQosGuaranteed(t, sidecar, true)

	// Check GS envs
	gameserver := &job.Spec.Template.Spec.Containers[1]
	checkEnvVar(t, gameserver, "MAP", string(data.Map))
	checkEnvVar(t, gameserver, "GAMEMODE", strconv.FormatInt(int64(data.GameMode), 10))
	// We need cpu affinity and QoS = guaranteed here
	//assertQosGuaranteed(t, gameserver, true)
	//assertCpuAffinity(t, gameserver, true)
	// Check that image is ok
	assertImage(t, gameserver, data.GameServerImage)
	assertResources(t, sidecar, sidecarResources)
	assertResources(t, gameserver, gameserverResources)
	assertPriorityClass(t, job)
	assertCommand(t, gameserver, gameserverCommand)
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
	//assertQosGuaranteed(t, sidecar, false)

	// Check GS envs
	gameserver := &job.Spec.Template.Spec.Containers[1]
	checkEnvVar(t, gameserver, "MAP", string(data.Map))
	checkEnvVar(t, gameserver, "GAMEMODE", strconv.FormatInt(int64(data.GameMode), 10))
	// We don't need cpu affinity and QoS = guaranteed here
	//assertQosGuaranteed(t, gameserver, false)
	//assertCpuAffinity(t, gameserver, false)
	// Check that image is ok
	assertImage(t, gameserver, data.GameServerImage)
	assertResources(t, sidecar, sidecarResources)
	assertResources(t, gameserver, gameserverResources)
	assertPriorityClass(t, job)
	assertCommand(t, gameserver, gameserverCommand)
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
	//preferred := affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	//for _, pref := range preferred {
	//	for _, expr := range pref.Preference.MatchExpressions {
	//		if wantVals, ok := expectedPreferred[expr.Key]; ok {
	//			if expr.Operator != corev1.NodeSelectorOpIn {
	//				t.Errorf("preferred key %s: operator = %s, want In", expr.Key, expr.Operator)
	//			}
	//			for _, val := range wantVals {
	//				if !contains(expr.Values, val) {
	//					t.Errorf("preferred key %s: missing value %s", expr.Key, val)
	//				}
	//			}
	//		}
	//	}
	//}
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

func assertResources(t *testing.T, container *corev1.Container, want corev1.ResourceRequirements) {
	t.Helper()
	if !apiequality.Semantic.DeepEqual(container.Resources, want) {
		t.Errorf("Container %s resources mismatch. Expected %v, got %v", container.Name, want, container.Resources)
	}
}

func assertPriorityClass(t *testing.T, job *batchv1.Job) {
	t.Helper()
	if got := job.Spec.Template.Spec.PriorityClassName; got != gamePriorityClass {
		t.Errorf("Pod priorityClassName mismatch. Expected %s, got %s", gamePriorityClass, got)
	}
}

func assertCommand(t *testing.T, container *corev1.Container, want []string) {
	t.Helper()
	if !reflect.DeepEqual(container.Command, want) || len(container.Args) != 0 {
		t.Errorf("Container %s command mismatch. Expected %q with no args, got %q args %q", container.Name, want, container.Command, container.Args)
	}
}

func assertImage(t *testing.T, container *corev1.Container, image string) {
	if container.Image != image {
		t.Errorf("Container image mismatch. Expected %s, got %s", image, container.Image)
	}
}
