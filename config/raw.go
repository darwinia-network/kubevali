package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type RawConfig struct {
	CommonTemplate string
	NodeTemplate   NodeTemplate
	NodeService    RawNodeService
	Watchlog       RawWatchlog
	NodeStdout     string
	NodeStderr     string
	Logging        interface{}
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

type RawNodeService struct {
	Enabled          bool
	NodePortTemplate string
	ForceUpdate      bool
}

func Unmarshal() *Config {
	raw := &RawConfig{}
	viper.Unmarshal(raw)
	validateOrDie(raw)
	return renderOrDie(raw)
}

func validateOrDie(raw *RawConfig) {
	if len := len(raw.NodeTemplate.Command); len < 1 {
		log.Fatalf("Config nodeTemplate.Command[] should at least have 1 element, got %d", len)
	}

	if raw.Watchlog.Enabled {
		if raw.Watchlog.Keyword == "" {
			log.Fatalf("Config watchlog.keyword should not be empty")
		}

		if raw.Watchlog.LastThreshold < 1*time.Second {
			log.Fatalf("Config watchlog.lastThreshold should be at least 1 second, got %s", raw.Watchlog.LastThreshold)
		}
	}
}
