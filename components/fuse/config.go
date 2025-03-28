package fuse

import (
	"database/sql"
	"encoding/json"

	query_config "github.com/leptonai/gpud/pkg/query/config"
)

type Config struct {
	Query query_config.Config `json:"query"`

	// CongestedPercentAgainstThreshold is the percentage of the FUSE connections waiting
	// at which we consider the system to be congested.
	CongestedPercentAgainstThreshold float64 `json:"congested_percent_against_threshold"`

	// MaxBackgroundPercentAgainstThreshold is the percentage of the FUSE connections waiting
	// at which we consider the system to be congested.
	MaxBackgroundPercentAgainstThreshold float64 `json:"max_background_percent_against_threshold"`
}

func ParseConfig(b any, dbRW *sql.DB, dbRO *sql.DB) (*Config, error) {
	raw, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	err = json.Unmarshal(raw, cfg)
	if err != nil {
		return nil, err
	}
	if cfg.Query.State != nil {
		cfg.Query.State.DBRW = dbRW
		cfg.Query.State.DBRO = dbRO
	}
	return cfg, nil
}

const (
	DefaultCongestedPercentAgainstThreshold     = float64(90)
	DefaultMaxBackgroundPercentAgainstThreshold = float64(80)
)

func (cfg *Config) Validate() error {
	if cfg.CongestedPercentAgainstThreshold == 0 {
		cfg.CongestedPercentAgainstThreshold = DefaultCongestedPercentAgainstThreshold
	}
	if cfg.MaxBackgroundPercentAgainstThreshold == 0 {
		cfg.MaxBackgroundPercentAgainstThreshold = DefaultMaxBackgroundPercentAgainstThreshold
	}
	return nil
}
