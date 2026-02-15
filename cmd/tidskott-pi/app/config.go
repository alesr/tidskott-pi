package app

import (
	"flag"
	"fmt"

	"github.com/alesr/tidskott-pi/internal/pkg/config"
)

type flags struct {
	ConfigPath string
}

func parseFlags() (*flags, error) {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()
	return &flags{ConfigPath: *configPath}, nil
}

func loadConfig(configPath string) (*config.Config, error) {
	var configFile string
	if configPath != "" {
		configFile = configPath
	} else {
		configFile = config.DefaultConfigPath()
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, nil
}
