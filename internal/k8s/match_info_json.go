package k8s

import (
	"encoding/json"

	"github.com/dota2classic/d2c-go-models/models"
)

type runServerSchema struct {
	MatchID      int64                  `json:"matchId"`
	LobbyType    models.MatchmakingMode `json:"lobbyType"`
	GameMode     models.DotaGameMode    `json:"gameMode"`
	RoomID       string                 `json:"roomId"`
	ServerURL    string                 `json:"serverUrl"`
	FillBots     bool                   `json:"fillBots"`
	EnableCheats bool                   `json:"enableCheats"`
	StrictPause  bool                   `json:"strictPause"`
	Players      []player               `json:"players"`
	Patch        models.DotaPatch       `json:"patch"`
	Region       models.Region          `json:"region"`
}

type player struct {
	SteamID    string          `json:"steamId"`
	Name       string          `json:"name"`
	Subscriber bool            `json:"subscriber"`
	Muted      bool            `json:"muted"`
	Ignore     bool            `json:"ignore"`
	PartyID    string          `json:"partyId"`
	Team       models.DotaTeam `json:"team"`
}

func constructMatchInfoJson(command *models.LaunchGameServerCommand) (string, error) {

	strictPause := command.LobbyType != models.MATCHMAKING_MODE_LOBBY && command.GameMode != models.DOTA_GAME_MODE_CAPTAINS_MODE

	var players []player
	for _, plr := range command.Players {
		players = append(players, player{
			SteamID:    plr.SteamID,
			Subscriber: plr.Subscriber,
			Name:       plr.Name,
			Muted:      plr.Muted,
			PartyID:    plr.PartyID,
			Team:       plr.Team,
			Ignore:     false,
		})
	}

	schema := runServerSchema{
		MatchID:      command.MatchID,
		LobbyType:    command.LobbyType,
		GameMode:     command.GameMode,
		RoomID:       command.RoomID,
		ServerURL:    "Deprecated",
		FillBots:     command.FillBots,
		EnableCheats: command.EnableCheats,
		Patch:        command.Patch,
		Region:       command.Region,
		StrictPause:  strictPause,
		Players:      players,
	}

	res, err := json.Marshal(schema)
	if err != nil {
		return "", err
	}

	return string(res), nil

}
