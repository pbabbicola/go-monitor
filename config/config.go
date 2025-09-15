package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

// SiteElement is a unit of configuration that describes the URL we need to monitor, the regexp that we want to check for, and the interval in which we should do so.
type SiteElement struct {
	URL             string         `json:"url"`
	Regexp          *regexp.Regexp `json:"regexp"`
	IntervalSeconds int            `json:"interval-seconds"`
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
