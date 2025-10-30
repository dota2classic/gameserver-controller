package util

type ServerInfo struct {
	URL       string `json:"url"`
	MatchId   int64  `json:"matchId"`
	Timestamp int64  `json:"timestamp"` // Unix timestamp in seconds
}
