package main

import (
	"crypto/tls"
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

/*
 * Use recommended TLS settings
 */
func setupTLSserver() *http.Server {
	cfg := &tls.Config{}

	// At least TLS v1.2 is recommended
	cfg.MinVersion = tls.VersionTLS12

	// Enable secure ciphers only
	for _, cipherSuite := range tls.CipherSuites() {
		cfg.CipherSuites = append(cfg.CipherSuites, cipherSuite.ID)
	}

	return &http.Server{
		Addr:      config.Host + ":" + config.Port,
		TLSConfig: cfg,
	}
}

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
	err = setupLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't setup a logger: %s", err.Error())
		os.Exit(1)
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
	 * Start an HTTPS server
	 */
	http.HandleFunc("/api", apiHandler)

	log.Info().Msgf("Graphoscope v%s. Starting the service listening on %s:%s", version, config.Host, config.Port)
	server := setupTLSserver()

	err = server.ListenAndServeTLS(config.CertFile, config.KeyFile)
	if err != nil {
		log.Fatal().Msg("Can't ListenAndServeTLS: " + err.Error())
	}
}
