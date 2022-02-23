package main

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

var (
	// Regex to remove "datetime" field from the query.
	// Useful when data source doesn't contain such field
	reDT = regexp.MustCompile(`(?i) +and +datetime +between +('|")\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?Z('|") +and +('|")\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d+)?Z('|")`)
)

/*
 * Parse SQL query for the later processing by the collectors,
 * textual SQL query into a logical object.
 *
 * Receives a query to parse, whether result should contain a "datetime" field,
 * whether data source supports SQL features
 */
func parseSQL(sql string, includeDatetime bool, replaceFields map[string]string, supportsSQL bool) ([]*sqlparser.Select, error) {

	// Remove "datetime" field from the query if must be ignored
	if !includeDatetime {
		sql = reDT.ReplaceAllString(sql, "")
	}

	/*
	 * Parse and validate received SQL query
	 */

	ast, err := sqlparser.Parse("SELECT * " + sql)
	if err != nil {
		return nil, fmt.Errorf("Can't parse SQL query: %s", err.Error())
	}

	query, ok := ast.(*sqlparser.Select)
	if !ok {
		return nil, fmt.Errorf("Only SELECT statement is allowed")
	}

	// Handle WHERE
	if query.Where == nil {
		return nil, fmt.Errorf("WHERE filters are missing")
	}

	switch query.Where.Expr.(type) {
	case *sqlparser.ParenExpr,
		*sqlparser.AndExpr,
		*sqlparser.OrExpr,
		*sqlparser.ComparisonExpr:
	default:
		return nil, fmt.Errorf("WHERE statement is not a list of filters")
	}

	// Handle LIMIT
	if query.Limit != nil {
		// Handle offset
		if query.Limit.Offset != nil {
			offset := sqlparser.String(query.Limit.Offset)

			_, err := strconv.Atoi(offset)
			if err != nil {
				return nil, fmt.Errorf("\"LIMIT ?,x\" value is not an integer: %v (%T)", offset, offset)
			}
		} else {
			query.Limit.Offset = &sqlparser.SQLVal{Type: 1, Val: []uint8{0x30}} // Start from 0
		}

		// Handle rowcount
		if query.Limit.Rowcount != nil {
			rowcount := sqlparser.String(query.Limit.Rowcount)

			limit, err := strconv.Atoi(rowcount)
			if err != nil {
				return nil, fmt.Errorf("\"LIMIT x,?\" value is not an integer: %v (%T)", rowcount, rowcount)
			}

			if limit < 0 {
				return nil, fmt.Errorf("LIMIT rowcount can't be less than 0")
			}

			// Decrease to the server-side value if user's custom value is too large
			if limit > config.Limit {
				query.Limit.Rowcount = &sqlparser.SQLVal{Type: 1, Val: []uint8(strconv.Itoa(config.Limit))}
			}
		}

	} else {
		query.Limit = &sqlparser.Limit{
			Offset:   &sqlparser.SQLVal{Type: 1, Val: []uint8{0x30}}, // Start from 0
			Rowcount: &sqlparser.SQLVal{Type: 1, Val: []uint8(strconv.Itoa(config.Limit))},
		}
	}

	// Handle multiple FROM
	if len(query.From) != 1 {
		return nil, fmt.Errorf("Multiple FROM currently not supported")
	}

	// Handle DISTINCT
	if query.Distinct != "" {
		return nil, fmt.Errorf("DISTINCT shouldn't be used, API service already returns unique nodes pairs only")
	}

	/*
	 * Split OR and IN queries into a list of separate queries
	 * for the data sources that don't support such queries directly
	 */

	queries := []*sqlparser.Select{}

	if !supportsSQL {
		queries, err = splitQuery(query)
		if err != nil {
			return nil, fmt.Errorf("Can't split query: " + err.Error())
		}
	} else {
		queries = append(queries, query)
	}

	/*
	 * Replace user defined fields
	 */

	for _, node := range queries {
		err = replaceSQL(node, replaceFields)
	}

	/*
	 * Return possibly modified result
	 */

	return queries, err
}

/*
 * Split clean OR and IN queries into a list of separate queries.
 *
 * Supported query types:
 *     - ip='8.8.8.8'
 *     - ip='8.8.8.8' OR ip='1.2.3.4'
 *     - domain IN ('example.com','google.com')
 *
 * Operators have to be "=" only, mix of different operators is not supported yet
 */
func splitQuery(expr *sqlparser.Select) ([]*sqlparser.Select, error) {

	where := expr.Where.Expr
	lefts := []sqlparser.Expr{}

	// A list of independend SELECT single-field queries to return
	selects := []*sqlparser.Select{}

	var err error

	//fmt.Printf("Expr: %#v\n", where)

	if and, ok := where.(*sqlparser.AndExpr); ok {
		if and.Right.(*sqlparser.RangeCond).Operator == "between" &&
			sqlparser.String(and.Right.(*sqlparser.RangeCond).Left) == "datetime" {

			// Handle:
			//     field='...' AND datetime BETWEEN ...
			//     field IN ('...','...') AND datetime BETWEEN ...
			if comp, ok := and.Left.(*sqlparser.ComparisonExpr); ok {
				lefts, err = splitOrIn(comp, nil)
				if err != nil {
					return nil, err
				}

				// Handle:
				//     (field='...') AND datetime BETWEEN ...
				//     (field='...' OR field='...') AND datetime BETWEEN ...
				//     (field IN ('...','...')) AND datetime BETWEEN ...
			} else if paren, ok := and.Left.(*sqlparser.ParenExpr); ok {
				lefts, err = splitOrIn(paren.Expr, nil)
				if err != nil {
					return nil, err
				}
			}
		}

		// Handle:
		//     (field='...' OR field=...')
		//     (field IN ('...','...'))
	} else if paren, ok := where.(*sqlparser.ParenExpr); ok {
		lefts, err = splitOrIn(paren.Expr, nil)
		if err != nil {
			return nil, err
		}

		// Handle: field='...' OR field=...'
	} else if or, ok := where.(*sqlparser.OrExpr); ok {
		lefts, err = splitOrIn(or, nil)
		if err != nil {
			return nil, err
		}

		// Handle:
		//     field='...'
		//     field IN ('...','...')
	} else if comp, ok := where.(*sqlparser.ComparisonExpr); ok {
		lefts, err = splitOrIn(comp, nil)
		if err != nil {
			return nil, err
		}
	}

	// Generate multiple independent single-field queries
	for _, left := range lefts {
		if and, ok := where.(*sqlparser.AndExpr); ok {
			and.Left = left
		} else if paren, ok := where.(*sqlparser.ParenExpr); ok {
			paren.Expr = left
		}

		// SQL -> string -> SQL
		// as it is hard to deep copy 'expr' to prevent rewriting it
		ast, err := sqlparser.Parse(sqlparser.String(expr))
		if err != nil {
			return nil, fmt.Errorf("Can't parse splitted SQL query: %s", err.Error())
		}

		query := ast.(*sqlparser.Select)
		selects = append(selects, query)
	}

	return selects, nil
}

/*
 * A loop to detect and split OR/IN query
 */
func splitOrIn(expr sqlparser.Expr, list []sqlparser.Expr) ([]sqlparser.Expr, error) {
	if list == nil {
		list = []sqlparser.Expr{}
	}

	var err error

	// Detect a clean OR query
	// and avoid more complex queries like "... OR (... AND ...)"
	switch left := expr.(type) {
	case *sqlparser.OrExpr:
		_, okOr := left.Left.(*sqlparser.OrExpr)
		comp, okComp := left.Left.(*sqlparser.ComparisonExpr)

		if okOr || (okComp && comp.Operator == "=") {
			if c, ok := left.Right.(*sqlparser.ComparisonExpr); ok {

				if c.Operator != "=" {
					return list, nil
				}

				list = append(list, left.Right)

				if c, ok := left.Left.(*sqlparser.ComparisonExpr); ok {
					list = append(list, c)

				} else if or, ok := left.Left.(*sqlparser.OrExpr); ok {
					list, err = splitOrIn(or, list)
					if err != nil {
						return nil, err
					}

				} else {
					return list, nil
				}
			}
		}

	// Detect a clean IN query.
	// Single 'field=value' can be represented as 1 element array
	case *sqlparser.ComparisonExpr:
		field := sqlparser.String(left.Left)

		if left.Operator == "in" {
			for _, value := range left.Right.(sqlparser.ValTuple) {
				ast, err := sqlparser.Parse("SELECT * WHERE " + field + "=" + sqlparser.String(value))
				if err != nil {
					return nil, err
				}

				list = append(list, ast.(*sqlparser.Select).Where.Expr)
			}

		} else if left.Operator == "=" {
			ast, err := sqlparser.Parse("SELECT * WHERE " + field + "=" + sqlparser.String(left.Right))
			if err != nil {
				return nil, err
			}

			list = append(list, ast.(*sqlparser.Select).Where.Expr)
		}

	default:
		log.Error().Msgf("Unexpected \"Left\" part of the query to split: %#v\n", left)
	}

	return list, nil
}
