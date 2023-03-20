package main

import (
	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Export symbols
 */
var (
	/*
	 * STEP 15.
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

	/*
	 * STEP 13.
	 *
	 * Inherit default configuration fields for the data source or
	 * processor plugin
	 */

	source *pdk.Source
	//processor *pdk.Processor

	/*
	 * STEP 14.
	 *
	 * Define all the custom fields needed by the plugin,
	 * such as "client" object, database/collection name, etc..
	 */

	//client *package.Client
	limit int
}
