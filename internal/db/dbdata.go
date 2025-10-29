package db

import "time"

type MatchResources struct {
	MatchId       int64
	JobName       string
	SecretName    string
	ConfigMapName string
	CreatedAt     time.Time
}
