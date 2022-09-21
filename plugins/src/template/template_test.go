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

	/*
	 * STEP 13.
	 *
	 * Test whether example SQL queries are correctly converted to the expected format.
	 * Sometimes it's easier to compare textual representation of the object,
	 * rather than all it's child objects & content
	 */

	// Pairs of SQLs and the expected results
	tables := []struct {
		sql       string
		converted string
	}{
		{`SELECT * WHERE ip='10.10.10.10' LIMIT 0,10`, ``},
		{`SELECT * WHERE size>100 ORDER BY name LIMIT 5,1`, ``},
		{`SELECT * WHERE size=10 ORDER BY name DESC LIMIT 0,1`, ``},
		{`SELECT * WHERE size>=100 LIMIT 0,1`, ``},
		{`SELECT * WHERE name LIKE 's%' LIMIT 0,1`, ``},
		{`SELECT * WHERE name NOT LIKE 's%' LIMIT 0,1`, ``},
		{`SELECT * WHERE size BETWEEN 100 AND 300 LIMIT 0,1`, ``},
		{`SELECT * WHERE size IN (100,300) LIMIT 0,1`, ``},
		{`SELECT * WHERE size NOT IN (100,300) LIMIT 0,1`, ``},
		{`SELECT * WHERE name='sarah' AND age!=40 AND (country='LV' OR country='AU') ORDER BY age DESC LIMIT 1`, ``},
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

		if result != table.converted {
			t.Errorf("Invalid conversion of \"%s\": \"%s\", expected: \"%s\"", table.sql, result, table.converted)
		}
	}
}
