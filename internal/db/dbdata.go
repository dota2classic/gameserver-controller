package db

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Status maps to Postgres match_status enum
type Status string

const (
	StatusLaunching Status = "launching"
	StatusRunning   Status = "running"
	StatusFailed    Status = "failed"
	StatusDone      Status = "done"
)

func (s *Status) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*s = Status(v)
		return nil
	case []byte:
		*s = Status(string(v))
		return nil
	default:
		return fmt.Errorf("failed to scan Status: unsupported type %T", value)
	}
}

func (s Status) Value() (driver.Value, error) {
	return string(s), nil
}

type MatchResources struct {
	MatchId       int64
	JobName       string
	SecretName    string
	ConfigMapName string
	CreatedAt     time.Time
	Status        Status
}

type GameServerSettings struct {
	MatchmakingMode int64
	TickRate        int
}
