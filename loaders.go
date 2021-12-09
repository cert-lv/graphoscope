package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	yaml "gopkg.in/yaml.v3"
)

var (
	// Query formatting rules,
	// which help to format comma/space separated indicators to a valid SQL query
	formats string
)

/*
 * Return content of the requested file by its path
 */
func loadFileIntoString(path string) (string, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(file), nil
}

/*
 * Load query formatting rules
 */
func loadFormats() error {
	buffer, err := loadFileIntoString(config.Formats)
	if err != nil {
		return fmt.Errorf("Failed to read rules file '%s': %s", config.Formats, err.Error())
	}

	var f map[string][]string

	err = yaml.Unmarshal([]byte(buffer), &f)
	if err != nil {
		return fmt.Errorf("Failed unmarshalling rules yaml: %s", err.Error())
	}

	// Validate regexps
	for group, res := range f {
		for _, re := range res {
			_, err = regexp.Compile(re)
			if err != nil {
				return fmt.Errorf("Invalid %s's regular expression '%s' : %s", group, re, err.Error())
			}
		}
	}

	b, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("Failed to marshal rules struct: %s", err.Error())
	}

	formats = string(b)
	return nil
}

/*
 * Load service's version
 */
func loadVersion() error {
	path := "VERSION"
	var err error

	// Try to get from the environment variable first
	if os.Getenv(path) != "" {
		version = os.Getenv(path)
		return nil
	}

	version, err = loadFileIntoString(path)
	if err != nil {
		return err
	}

	return nil
}
