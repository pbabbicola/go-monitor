package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"

	"github.com/caarlos0/env/v11"
)

// TODO: Add tests (running out of time) and use FileURL as an actual url.
// EnvConfig keeps the configuration parsed from the environment by [ParseEnv].
type EnvConfig struct {
	FileURL  string     `env:"FILE_URL" envDefault:"sample-big.json"`
	LogLevel slog.Level `env:"LOG_LEVEL" envDefault:"debug"`
}

// ParseEnv parses the configuration from the environment. If it fails, it returns a wrapped error from the env package.
func ParseEnv() (*EnvConfig, error) {
	envConfig := &EnvConfig{}

	err := env.Parse(envConfig)
	if err != nil {
		return nil, fmt.Errorf("parsing environment config: %w", err)
	}

	return envConfig, nil
}

// SiteElement is a unit of configuration that describes the URL we need to monitor, the regexp that we want to check for, and the interval in which we should do so.
type SiteElement struct {
	URL             string         `json:"url"`
	Regexp          *regexp.Regexp `json:"regexp"`
	IntervalSeconds int            `json:"interval_seconds"`
}

// Parse reads a filename, parses the json, and returns, if successful, a []SiteElement configuration.
//
// If it fails, it returns a wrapped error. Underlying errors can be from [regexp.Compile], [json.Unmarshal], or [os.ReadFile].
func Parse(filename string) ([]SiteElement, error) {
	fileContents, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file %v: %w", filename, err)
	}

	siteConfiguration := []SiteElement{}

	err = json.Unmarshal(fileContents, &siteConfiguration)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling file %v: %w", filename, err)
	}

	return siteConfiguration, nil
}
