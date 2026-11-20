package k8s

import (
	"bytes"
	"text/template"

	"github.com/dota2classic/d2c-go-models/models"
	"sigs.k8s.io/yaml"

	_ "embed"
)

//go:embed templates/secret.template.yaml
var SecretTemplate string

//go:embed templates/configmap.template.yaml
var ConfigmapTemplate string

//go:embed templates/cpu-affinity-job.template.yaml
var CpuAffinityJobTemplate string

//go:embed templates/regular-job.template.yaml
var RegularJobTemplate string

//const (
//	SECRET_TEMPLATE    = "./templates/secret.template.yaml"
//	CONFIGMAP_TEMPLATE = "./templates/configmap.template.yaml"
//	JOB_TEMPLATE       = "./templates/job.template.yaml"
//)

type templateData struct {
	MatchId         int64
	GameMode        models.DotaGameMode
	LobbyType       models.MatchmakingMode
	Map             models.DotaMap
	Region          models.Region
	RconPassword    string
	MatchJson       string
	GameServerImage string

	TickRate   int
	ConfigName string

	LoadTimeout int

	HostGamePort     int
	HostSourceTVPort int

	EnableBans         int
	DisableRunes       int
	MidTowerToWin      int
	KillsToWin         int
	AbandonHighQuality int

	BotDifficulty int
}

// GameServerCPURequest changes scheduling weight and reserved capacity, not a CPU cap.
func (d *templateData) GameServerCPURequest() string {
	switch d.LobbyType {
	case models.MATCHMAKING_MODE_BOTS, models.MATCHMAKING_MODE_BOTS_2X2,
		models.MATCHMAKING_MODE_SOLOMID, models.MATCHMAKING_MODE_TOURNAMENT_SOLOMID:
		return "250m"
	case models.MATCHMAKING_MODE_UNRANKED:
		if d.GameMode == models.DOTA_GAME_MODE_ALLPICK {
			return "1000m"
		}
	}
	return "425m"
}

func createConfiguration[T any](templateContent string, data *templateData) (*T, error) {
	tmpl, err := template.New("tmpl").Parse(templateContent)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	var config T
	if err = yaml.Unmarshal(buf.Bytes(), &config); err != nil {
		return nil, err
	}
	return &config, nil
}
