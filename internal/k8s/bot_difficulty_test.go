package k8s

import (
	"strconv"
	"testing"

	"github.com/dota2classic/d2c-go-models/models"
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
				job, err := renderJob(tmpl, &input)
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

func TestUnfairBotsPreserveExistingNiceTiers(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode models.MatchmakingMode
		tick int
		want string
	}{
		{"regularBots", models.MATCHMAKING_MODE_BOTS, 30, "15"},
		{"highTickKeepsImagePriority", models.MATCHMAKING_MODE_BOTS, 60, ""},
		{"bots2x2KeepsImagePriority", models.MATCHMAKING_MODE_BOTS_2X2, 30, ""},
		{"allpickKeepsImagePriority", models.MATCHMAKING_MODE_UNRANKED, 60, ""},
	} {
		for name, tmpl := range map[string]string{"regular": RegularJobTemplate, "affinity": CpuAffinityJobTemplate} {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				input := data
				input.LobbyType, input.TickRate = tc.mode, tc.tick
				input.BotDifficulty = botDifficultyForMode(tc.mode)
				job, err := renderJob(tmpl, &input)
				if err != nil {
					t.Fatal(err)
				}
				got := ""
				for _, env := range job.Spec.Template.Spec.Containers[1].Env {
					if env.Name == "SRCDS_NICE" {
						got = env.Value
					}
				}
				if got != tc.want {
					t.Fatalf("SRCDS_NICE = %q; want %q", got, tc.want)
				}
			})
		}
	}
}
