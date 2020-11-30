package config

import (
	"time"

	"github.com/sirupsen/logrus"
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

func Unmarshal() *Config {
	raw := &RawConfig{}
	viper.Unmarshal(raw)
	validateOrDie(raw)
	return renderOrDie(raw)
}

func validateOrDie(raw *RawConfig) {
	if len := len(raw.NodeTemplate.Command); len < 1 {
		logrus.Fatalf("Config nodeTemplate.Command[] should at least have 1 element, got %d", len)
	}

	if raw.Watchlog.Enabled {
		if raw.Watchlog.Keyword == "" {
			logrus.Fatalf("Config watchlog.keyword should not be empty")
		}

		if raw.Watchlog.LastThreshold < 1*time.Second {
			logrus.Fatalf("Config watchlog.lastThreshold should be at least 1 second, got %s", raw.Watchlog.LastThreshold)
		}
	}
}
