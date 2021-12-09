package main

import (
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Export symbols
var (
	Name    = "postgresql"
	Version = "1.0.0"
	Plugin  plugin
)

type plugin struct {

	// Inherit default configuration fields
	source *pdk.Source

	// Custom fields
	connection *pgxpool.Pool
	limit      int
}
