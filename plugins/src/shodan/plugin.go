package main

import (
	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Export symbols
 */
var (
	Name    = "shodan"
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
	limit   int
	pages   int
	credits int
}
