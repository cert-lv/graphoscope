/*
 * SQL to MySQL query convertor
 */

package main

import (
	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Convert SQL statement to the MySQL query
 */
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

	// Handle LIMIT offset,rowcount
	if sel.Limit != nil {
		if sel.Limit.Offset != nil {
			query += " LIMIT " + sqlparser.String(sel.Limit.Offset) + "," + sqlparser.String(sel.Limit.Rowcount)
		} else {
			query += " LIMIT 0," + sqlparser.String(sel.Limit.Rowcount)
		}
	}

	return query, nil
}
