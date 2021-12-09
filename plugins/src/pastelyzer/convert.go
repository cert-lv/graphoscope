/*
 * SQL to Pastelyzer query converter
 */

package main

import (
	"errors"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

// Convert SQL query to the Pastelyzer query
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

	// Handle group by
	if len(sel.GroupBy) > 0 || checkNeedAgg(sel.SelectExprs) {
		return nil, errors.New("'GROUP BY' & aggregation are not supported")
	}

	// Set limit
	if sel.Limit != nil {
		fields = append(fields, [2]string{"limit", sqlparser.String(sel.Limit.Rowcount)})
	}

	return fields, nil
}
