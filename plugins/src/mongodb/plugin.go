package main

import (
	"github.com/cert-lv/graphoscope/pdk"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
 * Export symbols
 */
var (
	Name    = "mongodb"
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
	client     *mongo.Client
	collection *mongo.Collection
	limit      int
}
