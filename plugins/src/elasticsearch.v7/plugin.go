package main

import (
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/olivere/elastic/v7"
)

/*
 * Export symbols
 */
var (
	Name    = "elasticsearch.v7"
	Version = "1.0.4"
	Plugin  plugin
)

/*
 * Structure to be imported by the core as a plugin
 */
type plugin struct {

	// Inherit default configuration fields
	source *pdk.Source

	// Custom fields
	client *elastic.Client
	index  string
	limit  int
}
