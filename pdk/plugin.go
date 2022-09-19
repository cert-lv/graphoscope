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

	// Get a list of all known data source's fields
	// for the Web GUI autocomplete
	Fields() ([]string, error)

	// Execute the given query.
	// Returns results, statistics, debug info & error
	Search(*sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error)

	// Stop the collector when the core service stops,
	// gracefully disconnect from the data source if needed
	Stop() error
}
