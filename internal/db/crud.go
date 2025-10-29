package db

func InsertMatchResources(mr MatchResources) error {
	db := ConnectAndMigrate()
	_, err := db.Exec(`INSERT INTO match_resources (match_id, job_name, secret_name, config_map_name) VALUES ($1, $2, $3, $4)`, mr.MatchId, mr.JobName, mr.SecretName, mr.ConfigMapName)
	return err
}

func FindMatchResources(id int64) (*MatchResources, error) {
	db := ConnectAndMigrate()
	row := db.QueryRow(`SELECT match_id, job_name, secret_name, config_map_name, created_at FROM match_resources WHERE match_id=$1`, id)
	var mr MatchResources
	if err := row.Scan(&mr.MatchId, &mr.JobName, &mr.SecretName, &mr.ConfigMapName, &mr.CreatedAt); err != nil {
		return nil, err
	}
	return &mr, nil
}
