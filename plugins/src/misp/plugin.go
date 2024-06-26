package main

import (
	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Export symbols
 */
var (
	Name    = "misp"
	Version = "1.0.0"
	Plugin  plugin
)

/*
 * Structure to be imported by the core as a plugin
 */
type plugin struct {

	// Inherit default configuration fields
	source *pdk.Source

	// Custom fields
	protocol   string
	host       string
	apiKey     string
	caCertPath string
	certPath   string
	keyPath    string
	types      map[string]bool
	limit      int
}
