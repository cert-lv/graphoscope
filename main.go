package main

import (
	"fmt"
	"net/http"
	"os"

	//_ "net/http/pprof"

	"github.com/rs/zerolog"
)

var (
	// Holder all service's configuration
	config *Config

	// Instance of the global logger
	log zerolog.Logger

	// Current service's version
	version string
)

func main() {
	/*
	 * Parse configuration file
	 */
	err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't load configuration: %s", err.Error())
		os.Exit(1)
	}

	/*
	 * Setup a global logger to the file or stdout
	 */
	fp, err := setupLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't setup a logfile: %s", err.Error())
		os.Exit(1)
	}
	if fp != nil {
		defer fp.Close()
	}

	/*
	 * Setup a database
	 */
	err = setupDatabase()
	if err != nil {
		log.Fatal().Msg("Can't setup a database: " + err.Error())
	}

	/*
	 * Load plugins
	 */
	err = loadPlugins()
	if err != nil {
		log.Fatal().Msg("Can't load plugins: " + err.Error())
	}

	/*
	 * Setup collectors for the predefined data sources
	 */
	err = setupCollectors()
	if err != nil {
		log.Fatal().Msg("Can't load collectors: " + err.Error())
	}

	/*
	 * Stop collectors on service exit
	 */
	defer func() {
		for name, collector := range collectors {
			err := collector.Stop()

			if err != nil {
				log.Error().
					Str("source", name).
					Msg("Can't stop the collector: " + err.Error())
			} else {
				log.Debug().
					Str("source", name).
					Msg("Collector stopped")
			}
		}
	}()

	// Load service's version
	err = loadVersion()
	if err != nil {
		log.Fatal().Msg("Can't load version: " + err.Error())
	}

	/*
	 * Start a Web GUI if needed
	 */
	err = startGUI()
	if err != nil {
		log.Fatal().Msg("Can't start Web GUI components: " + err.Error())
	}

	/*
	 * Start an API feature
	 */
	http.HandleFunc("/api", apiHandler)

	log.Info().Msgf("Graphoscope v%s. Starting the service listening on %s:%s", version, config.Host, config.Port)
	err = http.ListenAndServeTLS(config.Host+":"+config.Port, config.CertFile, config.KeyFile, nil)
	if err != nil {
		log.Fatal().Msg("Can't ListenAndServeTLS: " + err.Error())
	}
}
