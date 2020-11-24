package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type RawConfig struct {
	CommonTemplate string
	NodeTemplate   NodeTemplate
	Watchlog       RawWatchlog
}

type RawWatchlog struct {
	Enabled        bool
	Keyword        string
	LastThreshold  time.Duration
	HealthcheckIDs []string
}

type NodeTemplate struct {
	Index   string
	Command []string
	Args    map[string]string
}

func NewConfig(path string) *Config {
	viper.SetConfigFile(path)

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Unable to load config file: %s", err))
	}

	raw := &RawConfig{}
	viper.Unmarshal(raw)

	if err := validate(raw); err != nil {
		panic(err)
	}

	return renderOrDie(raw)
}

func validate(raw *RawConfig) error {
	if len := len(raw.NodeTemplate.Command); len < 1 {
		return fmt.Errorf("Config nodeTemplate.Command[] should at least have 1 element, got %d", len)
	}

	if raw.Watchlog.Enabled {
		if raw.Watchlog.Keyword == "" {
			return fmt.Errorf("Config watchlog.keyword should not be empty")
		}

		if raw.Watchlog.LastThreshold < 1*time.Second {
			return fmt.Errorf("Config watchlog.lastThreshold should be at least 1 second, got %s", raw.Watchlog.LastThreshold)
		}
	}

	return nil
}
