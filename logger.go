package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

/*
 * Setup logger to the file and stdout.
 *
 * In a production environment log events to the file only,
 * in a development environment log to the stdout only
 */
func setupLogger() error {

	// For the production server
	if config.Environment == "prod" {
		// Lumberjack provides log files rotation
		log = zerolog.New(&lumberjack.Logger{
			Filename:   config.Log.File,
			MaxSize:    config.Log.MaxSize,    // Size in MB before file gets rotated
			MaxBackups: config.Log.MaxBackups, // Max number of files kept before being overwritten
			MaxAge:     config.Log.MaxAge,     // Max number of days to keep the files
			Compress:   true,                  // Whether to compress log files using gzip
		}).With().Timestamp().Logger()

		zerolog.SetGlobalLevel(config.Log.Level)

		return nil
	}

	// For the development
	stdout := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	log = zerolog.New(stdout).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(config.Log.Level)

	return nil
}
