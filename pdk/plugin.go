package pdk

import (
	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Plugin interface to be implemented by the data source plugins
 */
type SourcePlugin interface {
	// Return data source instance configuration
	Conf() *Source

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

/*
 * Plugin interface to be implemented by the processor plugins
 */
type ProcessorPlugin interface {
	// Return instance configuration
	Conf() *Processor

	// Set specific parameters for the instance,
	// establish connection, etc.
	Setup(*Processor) error

	// Process data source's received data in a background
	Process([]map[string]interface{}) ([]map[string]interface{}, error)

	// Stop the processor when the core service stops,
	// gracefully disconnect if needed
	Stop() error
}
