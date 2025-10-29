package k8s

import (
	"bytes"
	"text/template"

	"github.com/dota2classic/d2c-go-models/models"
	"sigs.k8s.io/yaml"
)

const (
	SECRET_TEMPLATE    = "templates/secret.template.yaml"
	CONFIGMAP_TEMPLATE = "templates/configmap.template.yaml"
	JOB_TEMPLATE       = "templates/job.template.yaml"
)

type templateData struct {
	MatchId      int64
	GameMode     models.DotaGameMode
	LobbyType    models.MatchmakingMode
	Map          models.DotaMap
	Region       models.Region
	RconPassword string
	MatchJson    string
}

func createConfiguration[T any](templatePath string, data *templateData) (*T, error) {
	tmpl, err := template.ParseFiles(templatePath)
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
