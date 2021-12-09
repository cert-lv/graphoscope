package main

import (
	"fmt"

	yaml "gopkg.in/yaml.v3"
)

var (
	// A list of new features for the current service's version.
	// Will be displayed once for each user
	features = []string{}
)

/*
 * Load new features from a YAML file
 */
func loadFeatures() error {
	buffer, err := loadFileIntoString(config.Features)
	if err != nil {
		return fmt.Errorf("Failed to read new features file '%s': %s", config.Features, err.Error())
	}

	err = yaml.Unmarshal([]byte(buffer), &features)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling new features yaml: %s", err.Error())
	}

	if len(features) > 0 {
		log.Debug().Msgf("New features loaded: %v", features)
	}

	return nil
}

/*
 * Prevent future notifications after the first one
 */
func (a *Account) hideFeatures() {
	if a.SeenFeatures == features[0] {
		log.Debug().
			Str("username", a.Username).
			Msg("No new features notifications to disable")
		return
	}

	err := a.update("seenFeatures", features[0])
	if err != nil {
		log.Error().Msg("Can't update account to hide new features notifications: " + err.Error())
		return
	}

	log.Debug().
		Str("username", a.Username).
		Msg("New features notifications are hidden")
}
