package db

import (
	"log"

	"github.com/dota2classic/d2c-go-models/models"
)

func InsertMatchResources(mr MatchResources) error {
	db := ConnectAndMigrate()
	_, err := db.Exec(`INSERT INTO match_resources (match_id, job_name, secret_name, config_map_name) VALUES ($1, $2, $3, $4)`, mr.MatchId, mr.JobName, mr.SecretName, mr.ConfigMapName)
	return err
}

func FindMatchResources(id int64) (*MatchResources, error) {
	db := ConnectAndMigrate()
	row := db.QueryRow(`SELECT match_id, job_name, secret_name, config_map_name, created_at, status FROM match_resources WHERE match_id=$1`, id)
	var mr MatchResources
	if err := row.Scan(&mr.MatchId, &mr.JobName, &mr.SecretName, &mr.ConfigMapName, &mr.CreatedAt, &mr.Status); err != nil {
		return nil, err
	}
	return &mr, nil
}

func FindAllMatchResources() ([]MatchResources, error) {
	db := ConnectAndMigrate()
	rows, err := db.Query(`
        SELECT match_id, job_name, secret_name, config_map_name, created_at, status
        FROM match_resources
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []MatchResources

	for rows.Next() {
		var mr MatchResources
		if err := rows.Scan(&mr.MatchId, &mr.JobName, &mr.SecretName, &mr.ConfigMapName, &mr.CreatedAt, &mr.Status); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		resources = append(resources, mr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return resources, nil
}

func UpdateStatus(matchId int64, status Status) error {
	db := ConnectAndMigrate()
	rows, err := db.Query("UPDATE match_resources SET status = $1 WHERE match_id=$2", status, matchId)
	if err != nil {
		log.Printf("Failed to update status: %v", err)
		return err
	}
	defer rows.Close()
	return nil
}

func DeleteMatchResources(matchId int64) {
	db := ConnectAndMigrate()
	rows, err := db.Query("DELETE FROM match_resources WHERE match_id=$1", matchId)
	if err != nil {
		log.Printf("Failed to delete resources: %v", err)
	}
	defer rows.Close()
}

func GetSettingsForMode(mode models.MatchmakingMode) (*GameServerSettings, error) {
	db := ConnectAndMigrate()
	row := db.QueryRow(`SELECT matchmaking_mode, tickrate, image, load_timeout, cpu_affinity FROM gameserver_settings WHERE matchmaking_mode=$1`, mode)
	var gss GameServerSettings
	if err := row.Scan(&gss.MatchmakingMode, &gss.TickRate, &gss.Image, &gss.LoadTimeout, &gss.CpuAffinity); err != nil {
		return nil, err
	}
	return &gss, nil
}
