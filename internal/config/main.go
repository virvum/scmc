package config

import (
	"fmt"
	"io/ioutil"

	"github.com/virvum/scmc/pkg/logger"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Username string
	Password string
	LogLevel logger.Level
}

func Load(configFile string, log *logger.Log) (*Config, error) {
	var cfg Config

	// Default values
	cfg.LogLevel = logger.Warn

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile(%s): %v", configFile, err)
	}

	if err = yaml.UnmarshalStrict(data, &cfg); err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal: %v", err)
	}

	log.Debug("%s successfully loaded", configFile)

	return &cfg, nil
}
