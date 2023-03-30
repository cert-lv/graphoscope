package main

import (
	"database/sql"

	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Export symbols
 */
var (
	Name    = "file-csv"
	Version = "1.0.5"
	Plugin  plugin
)

/*
 * Structure to be imported by the core as a plugin
 */
type plugin struct {

	// Inherit default configuration fields
	source *pdk.Source

	// Custom fields
	db    *sql.DB
	dir   string
	base  string
	limit int
}
