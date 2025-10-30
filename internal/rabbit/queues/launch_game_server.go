package queues

import (
	"context"
	"d2c-gs-controller/internal/db"
	"d2c-gs-controller/internal/k8s"
	"log"

	"github.com/dota2classic/d2c-go-models/models"
)

func HandleLaunchGameServerCommand(event *models.LaunchGameServerCommand) error {
	log.Printf("Launching game server for matchId %d", event.MatchID)
	mr, err := k8s.DeployMatchResources(context.Background(), k8s.GetClient(), event)
	if err != nil {
		log.Printf("Failed to deploy match: %v", err)
		return err
	}
	log.Printf("Match %d successfully deployed", event.MatchID)
	err = db.InsertMatchResources(db.MatchResources{
		MatchId:       event.MatchID,
		JobName:       mr.JobName,
		SecretName:    mr.SecretName,
		ConfigMapName: mr.ConfigMapName,
	})
	if err != nil {
		log.Printf("Failed to insert match: %v", err)
		return err
	}
	return nil
}
