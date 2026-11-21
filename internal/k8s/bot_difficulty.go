package k8s

import "github.com/dota2classic/d2c-go-models/models"

// The image infers nice +15 from BOT_DIFFICULTY=2. Preserve that
// scheduling tier when regular bot matches switch to Unfair (4).
func (d *templateData) GameServerNiceOverride() string {
	if d.LobbyType == models.MATCHMAKING_MODE_BOTS && d.TickRate < 60 {
		return "15"
	}
	return ""
}

func botDifficultyForMode(mode models.MatchmakingMode) int {
	switch mode {
	case models.MATCHMAKING_MODE_BOTS, models.MATCHMAKING_MODE_BOTS_2X2:
		return 4 // Unfair in the Source 1 Dota bot difficulty scale.
	default:
		return 3
	}
}
