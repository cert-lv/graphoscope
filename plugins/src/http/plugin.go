package main

import (
	"github.com/cert-lv/graphoscope/pdk"
)

// Export symbols
var (
	Name    = "http"
	Version = "1.0.0"
	Plugin  plugin
)

type plugin struct {

	// Inherit default configuration fields
	source *pdk.Source

	// Custom fields
	url    string
	method string
	limit  int
}
