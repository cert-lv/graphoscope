/*
 * SQL to the field/value list convertor
 */

package main

import (
	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Convert SQL query to the list of [field,value]
 */
func (p *plugin) convert(sel *sqlparser.Select) ([][2]string, error) {

	// Handle WHERE.
	// Top level node pass in an empty interface
	// to tell the children this is root.
	// Is there any better way?
	var rootParent sqlparser.Expr

	// List of requested fields & values
	fields, err := handleSelectWhere(&sel.Where.Expr, true, &rootParent)
	if err != nil {
		return nil, err
	}

	for i, field := range fields {
		if field[0] != "ipAddress" {
			fields = append(fields[:i], fields[i+1:]...)
		}
	}

	return fields, nil
}
