package main

import (
	"testing"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

// Test SQL conversion to the data source's expected format
func TestConvert(t *testing.T) {

	// Empty plugin's instance to test
	c := plugin{}

	// Pairs of SQLs and the expected results
	tables := []struct {
		sql       string
		converted [][2]string
	}{
		{`SELECT * WHERE ip='10.10.10.10' LIMIT 5,1`, [][2]string{[2]string{"ip", "10.10.10.10"}, [2]string{"offset", "5"}, [2]string{"rowcount", "1"}}},
		{`SELECT * WHERE size BETWEEN 100 AND 300`, [][2]string{[2]string{"size_from", "100"}, [2]string{"size_to", "300"}}},
		{`select * where name='sarah' and age=40 limit 1`, [][2]string{[2]string{"name", "sarah"}, [2]string{"age", "40"}, [2]string{"operator", "and"}, [2]string{"offset", "0"}, [2]string{"rowcount", "1"}}},
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

// equal tells whether a and b slices contain the same elements
func equal(a, b [][2]string) bool {
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
