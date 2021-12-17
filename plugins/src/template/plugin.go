package main

import (
	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Export symbols
 */
var (
	/*
	 * STEP 10.
	 *
	 * Set plugin name and version
	 */

	Name    = "template"
	Version = "1.0.0"
	Plugin  plugin
)

/*
 * Structure to be imported by the core as a plugin
 */
type plugin struct {

	// Inherit default configuration fields
	source *pdk.Source

	/*
	 * STEP 9.
	 *
	 * Define all the custom fields needed by the plugin,
	 * such as "client" object, database/collection name, etc..
	 */

	//client *package.Client
	limit int
}
