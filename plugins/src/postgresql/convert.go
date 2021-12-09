/*
 * SQL to PostgreSQL query converter
 */

package main

import (
	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

// convert SQL statement to the PostgreSQL filter & options
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
		// Handle offset
		if sel.Limit.Offset != nil {
			query += " OFFSET " + sqlparser.String(sel.Limit.Offset)
		} else {
			query += " OFFSET 0"
		}

		// Handle rowcount
		query += " LIMIT " + sqlparser.String(sel.Limit.Rowcount)
	}

	// Replace ` chars around keywords as PostgreSQL requires.
	//
	// TODO: Find cases when SQL parser produces backtiks
	// and replace them in a correct way. Simple "strings.Replace" will replace
	// expected and valid chars in the middle of the string too.
	//
	//query = strings.Replace(query, "`", "\"", -1)

	return query, nil
}
