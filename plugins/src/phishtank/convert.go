/*
 * SQL to the field/value list convertor
 */

package main

import (
	"net/url"
	"strings"

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

	// Requested field & value
	field, err := handleSelectWhere(&sel.Where.Expr, true, &rootParent)
	if err != nil {
		return nil, err
	}

	var fields [][2]string

	urlParsed, err := url.Parse(field[1])
	if err != nil {
		return nil, err
	}

	// Phishtank may contain "/" at the end of a domain and may not contain,
	// but it is very important for the search, so we include both variants
	if field[0] == "url" {
		fields = append(fields, [2]string{field[0], field[1]})

		if urlParsed.Path == "" {
			fields = append(fields, [2]string{field[0], field[1] + "/"})
		} else if urlParsed.Path == "/" {
			fields = append(fields, [2]string{field[0], strings.TrimRight(field[1], "/")})
		}
	}

	// Allow to search for a domain from Web GUI
	if field[0] == "domain" {
		fields = append(fields, [2]string{"url", "https://" + field[1] + "/"})
		fields = append(fields, [2]string{"url", "https://" + field[1]})
		fields = append(fields, [2]string{"url", "http://" + field[1] + "/"})
		fields = append(fields, [2]string{"url", "http://" + field[1]})
	}

	return fields, nil
}
