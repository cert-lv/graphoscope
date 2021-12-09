/*
 * SQL to CSV query converter
 */

package main

import (
	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

// convert SQL statement to the CSV query
func (p *plugin) convert(sel *sqlparser.Select) (string, error) {

	// Handle WHERE
	query := sqlparser.String(sel.Where.Expr)

	// Handle GROUP BY
	if len(sel.GroupBy) > 0 {
		query += sqlparser.String(sel.GroupBy)
	}

	// Handle ORDER BY
	if sel.OrderBy != nil {
		query += sqlparser.String(sel.OrderBy)
	}

	// Handle limit
	if sel.Limit != nil {
		// Handle rowcount
		query += " LIMIT " + sqlparser.String(sel.Limit.Rowcount)
		// Handle offset
		if sel.Limit.Offset != nil {
			query += " OFFSET " + sqlparser.String(sel.Limit.Offset)
		} else {
			query += " OFFSET 0"
		}
	}

	return query, nil
}
