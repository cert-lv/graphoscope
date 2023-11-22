package main

import (
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/redis/go-redis/v9"
)

/*
 * Export symbols
 */
var (
	Name    = "redis"
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
	client *redis.Client
	limit  int
}
