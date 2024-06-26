package main

import (
	"testing"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Test SQL conversion to the data source's expected format
 */
func TestConvert(t *testing.T) {

	// Empty plugin's instance to test
	c := plugin{}

	// Pairs of SQLs and the expected results
	tables := []struct {
		sql       string
		converted []string
	}{
		{
			"SELECT * WHERE `domain|ip`='10.10.10.10' LIMIT 5,1",
			[]string{"domain|ip", "10.10.10.10"},
		},
		{
			`SELECT * WHERE ip='10.10.10.10' and datetime BETWEEN '2024-05-04T11:30:14.000Z' AND '2024-06-04T11:30:14.000Z'`,
			[]string{"ip", "10.10.10.10", "2024-05-04T11:30:14.000Z", "2024-06-04T11:30:14.000Z"},
		},
		{
			`select * where name='sarah'`,
			[]string{"name", "sarah"},
		},
	}

	for _, table := range tables {
		// Executed by the main service
		ast, err := sqlparser.Parse(table.sql)
		if err != nil {
			t.Errorf("Can't parse '%s': %s", table.sql, err.Error())
			continue
		}

		stmt, ok := ast.(*sqlparser.Select)
		if !ok {
			t.Errorf("Only SELECT statement is allowed: %s", table.sql)
			continue
		}

		// Executed by the plugin
		result, err := c.convert(stmt)
		if err != nil {
			t.Errorf("Can't convert '%s': %s", table.sql, err.Error())
			continue
		}

		if !equal(result, table.converted) {
			t.Errorf("Invalid conversion of '%s': %v, expected: %v", table.sql, result, table.converted)
		}
	}
}

/*
 * Check whether a and b slices contain the same elements
 */
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}
