package main

import (
	"github.com/0x63616c/screenspace/server/internal/config"
)

// Config is re-exported from internal/config for use in main.
type Config = config.Config

// LoadConfig loads server config from environment variables.
func LoadConfig() (*Config, error) {
	return config.Load()
}
