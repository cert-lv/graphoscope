package main

import (
	"fmt"
	"testing"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Test SQL conversion to the data source's expected format
 */
func TestConvert(t *testing.T) {

	// Empty plugin's instance to test
	p := plugin{}

	p.source = &pdk.Source{
		Access: map[string]string{
			"field": "email",
		},
	}

	// Pairs of SQLs and the expected results
	tables := []struct {
		sql       string
		converted string
	}{
		{`SELECT * WHERE email='a@example.com'`, "a@example.com"},
		{`SELECT * WHERE user='a@example.com'`, ""},
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
		fmt.Println(1, p.source)
		result, err := p.convert(stmt)
		if err != nil {
			t.Errorf("Can't convert \"%s\": %s", table.sql, err.Error())
			continue
		}

		if result != table.converted {
			t.Errorf("Invalid conversion of \"%s\": %v, expected: %v", table.sql, result, table.converted)
		}
	}
}
