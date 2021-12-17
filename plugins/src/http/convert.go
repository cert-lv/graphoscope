/*
 * SQL to the field/value list convertor
 */

package main

import (
	"errors"

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

	// Handle GROUP BY
	if len(sel.GroupBy) > 0 || checkNeedAgg(sel.SelectExprs) {
		return nil, errors.New("'GROUP BY' & aggregation are not supported")
	}

	// Set selection OFFSET and ROWCOUNT
	if sel.Limit != nil {
		if sel.Limit.Offset != nil {
			fields = append(fields, [2]string{"offset", sqlparser.String(sel.Limit.Offset)})
		} else {
			fields = append(fields, [2]string{"offset", "0"})
		}

		fields = append(fields, [2]string{"rowcount", sqlparser.String(sel.Limit.Rowcount)})
	}

	return fields, nil
}
