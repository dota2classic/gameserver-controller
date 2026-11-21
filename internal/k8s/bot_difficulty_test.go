package k8s

import (
	"strconv"
	"testing"

	"github.com/dota2classic/d2c-go-models/models"
	batchv1 "k8s.io/api/batch/v1"
)

func TestBotDifficultyInBothJobTemplates(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode models.MatchmakingMode
		want int
	}{
		{"bots", models.MATCHMAKING_MODE_BOTS, 4},
		{"bots2x2", models.MATCHMAKING_MODE_BOTS_2X2, 4},
		{"allpick", models.MATCHMAKING_MODE_UNRANKED, 3},
		{"privateLobby", models.MATCHMAKING_MODE_LOBBY, 3},
		{"turbo", models.MATCHMAKING_MODE_TURBO, 3},
	} {
		for name, tmpl := range map[string]string{"regular": RegularJobTemplate, "affinity": CpuAffinityJobTemplate} {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				input := data
				input.LobbyType = tc.mode
				input.BotDifficulty = botDifficultyForMode(tc.mode)
				job, err := createConfiguration[batchv1.Job](tmpl, &input)
				if err != nil {
					t.Fatal(err)
				}
				for _, env := range job.Spec.Template.Spec.Containers[1].Env {
					if env.Name == "BOT_DIFFICULTY" {
						if env.Value != strconv.Itoa(tc.want) {
							t.Fatalf("BOT_DIFFICULTY = %q; want %d", env.Value, tc.want)
						}
						return
					}
				}
				t.Fatal("BOT_DIFFICULTY missing from gameserver environment")
			})
		}
	}
}
