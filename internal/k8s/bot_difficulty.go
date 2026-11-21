package k8s

import "github.com/dota2classic/d2c-go-models/models"

func botDifficultyForMode(mode models.MatchmakingMode) int {
	switch mode {
	case models.MATCHMAKING_MODE_BOTS, models.MATCHMAKING_MODE_BOTS_2X2:
		return 4 // Unfair in the Source 1 Dota bot difficulty scale.
	default:
		return 3
	}
}
