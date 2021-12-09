package pdk

import (
	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Plugin interface to be implemented by the plugins
 */
type Plugin interface {

	// Return data source instance definition
	Source() *Source
	// Set specific parameters for the data source instance,
	// establish connection, etc.
	Setup(*Source, int) error
	// Execute the given query.
	// Returns results, statistics & error
	Search(*sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, error)
	// Stop the collector when the core service stops,
	// gracefully disconnect from the data source if needed
	Stop() error
}
