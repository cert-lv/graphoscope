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
		converted string
	}{
		{`SELECT * WHERE ip='10.10.10.10'`, `ip = '10.10.10.10'`},
		{`SELECT * WHERE ip='10.10.10.10' LIMIT 5,10`, `ip = '10.10.10.10' LIMIT 5,10`},
		{`SELECT * WHERE size>100 ORDER BY name LIMIT 0,1`, `size > 100 order by name asc LIMIT 0,1`},
		{`SELECT * WHERE size=10 ORDER BY name DESC LIMIT 0,1`, `size = 10 order by name desc LIMIT 0,1`},
		{`SELECT * WHERE size>=100`, `size >= 100`},
		{`SELECT * WHERE name LIKE 's%'`, `name like 's%'`},
		{`SELECT * WHERE name NOT LIKE 's%'`, `name not like 's%'`},
		{`SELECT * WHERE size BETWEEN 100 AND 300`, `size between 100 and 300`},
		{`SELECT * WHERE size IN (100,300)`, `size in (100, 300)`},
		{`SELECT * WHERE size NOT IN (100,300)`, `size not in (100, 300)`},
		{`SELECT * WHERE name='sarah' and age!=40 AND (country='LV' OR country='AU') order by age desc limit 1`,
			`name = 'sarah' and age != 40 and (country = 'LV' or country = 'AU') order by age desc LIMIT 0,1`},
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
