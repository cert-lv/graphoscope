package main

import (
	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Export symbols
 */
var (
	Name    = "abuseipdb"
	Version = "1.0.1"
	Plugin  plugin
)

/*
 * Structure to be imported by the core as a plugin
 */
type plugin struct {

	// Inherit default configuration fields
	source *pdk.Source

	// Custom fields
	url          string
	maxAgeInDays int
	key          string
	limit        int
}
