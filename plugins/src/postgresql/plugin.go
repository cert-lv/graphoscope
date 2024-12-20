package main

import (
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/jackc/pgx/v4/pgxpool"
)

/*
 * Export symbols
 */
var (
	Name    = "postgresql"
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
	connection *pgxpool.Pool
	limit      int
}
