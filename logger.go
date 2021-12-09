package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

/*
 * Setup logger to the file and stdout.
 *
 * In a production environment log events to the file only,
 * in a development environment log to the stdout only
 */
func setupLogger() (*os.File, error) {

	// For the production server
	if config.Environment == "prod" {
		file, err := os.OpenFile(config.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, err
		}

		log = zerolog.New(file).With().Timestamp().Logger()
		zerolog.SetGlobalLevel(config.Log.Level)

		return file, nil
	}

	// For the development
	stdout := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	log = zerolog.New(stdout).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(config.Log.Level)

	return nil, nil
}
