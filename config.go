package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	yaml "gopkg.in/yaml.v3"
)

/*
 * Structure to store all the service settings.
 * Check "graphoscope.yaml.example" file for a detailed all fields description
 */
type Config struct {
	Server *struct {
		Host              string `yaml:"host"`
		Port              string `yaml:"port"`
		CertFile          string `yaml:"certFile"`
		KeyFile           string `yaml:"keyFile"`
		ReadTimeout       int    `yaml:"readTimeout"`
		ReadHeaderTimeout int    `yaml:"readHeaderTimeout"`
	} `yaml:"server"`

	Environment       string `yaml:"environment"`
	Definitions       string `yaml:"definitions"`
	Plugins           string `yaml:"plugins"`
	Limit             int    `yaml:"limit"`
	StabilizationTime int    `yaml:"stabilizationTime"`

	Log *struct {
		File       string        `yaml:"file"`
		MaxSize    int           `yaml:"maxSize"`
		MaxBackups int           `yaml:"maxBackups"`
		MaxAge     int           `yaml:"maxAge"`
		Level      zerolog.Level `yaml:"level"`
	} `yaml:"log"`

	Upload *struct {
		Path             string `yaml:"path"`
		MaxSize          int64  `yaml:"maxSize"`
		MaxIndicators    int    `yaml:"maxIndicators"`
		DeleteInterval   int    `yaml:"deleteInterval"`
		DeleteExpiration int    `yaml:"deleteExpiration"`
	} `yaml:"upload"`

	Groups   string `yaml:"groups"`
	Formats  string `yaml:"formats"`
	Features string `yaml:"features"`
	Docs     string `yaml:"docs"`

	Database struct {
		URL        string `yaml:"url"`
		Name       string `yaml:"name"`
		User       string `yaml:"user"`
		Password   string `yaml:"password"`
		Users      string `yaml:"users"`
		Dashboards string `yaml:"dashboards"`
		Notes      string `yaml:"notes"`
		Sessions   string `yaml:"sessions"`
		Cache      string `yaml:"cache"`
		Settings   string `yaml:"settings"`
		Timeout    int    `yaml:"timeout"`
		CacheTTL   int32  `yaml:"cacheTTL"`
	} `yaml:"database"`

	Sessions *struct {
		TTL               int    `yaml:"ttl"`
		CookieName        string `yaml:"cookieName"`
		AuthenticationKey string `yaml:"authenticationKey"`
		EncryptionKey     string `yaml:"encryptionKey"`
	} `yaml:"sessions"`
}

/*
 * Load configuration from a YAML file.
 *
 * Service searches for the "./graphoscope.yaml" file by default.
 * however, "CONFIG" environment variable can be set to use a different file
 */
func loadConfig() error {
	path := "graphoscope.yaml"

	if os.Getenv("CONFIG") != "" {
		path = os.Getenv("CONFIG")
	}

	buffer, err := loadFileIntoString(path)
	if err != nil {
		return fmt.Errorf("Failed to open configuration file '%s': %s", path, err.Error())
	}

	err = yaml.Unmarshal([]byte(buffer), &config)
	if err != nil {
		return fmt.Errorf("Invalid configuration YAML file '%s': %s", path, err.Error())
	}

	return nil
}
