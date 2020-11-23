package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	CommonTemplate string
	NodeTemplate   NodeTemplate
	Watchlog       Watchlog
}

type Watchlog struct {
	Enabled       bool
	HealthcheckID string
	Keyword       string
	LastThreshold time.Duration
}

type NodeTemplate struct {
	Command []string
	Args    map[string]string
}

func NewConfig() *Config {
	viper.SetConfigName("kubevali")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Unable to load config file: %s", err))
	}

	conf := Config{}
	viper.Unmarshal(&conf)

	return &conf
}
