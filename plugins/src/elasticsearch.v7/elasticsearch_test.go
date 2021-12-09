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
		converted string
	}{
		{`SELECT * WHERE ip='10.10.10.10'`, `{"query" : {"bool" : {"must" : [{"match_phrase" : {"ip" : "10.10.10.10"}}]}}}`},
		{`SELECT * WHERE ip='10.10.10.10' LIMIT 5,10`, `{"query" : {"bool" : {"must" : [{"match_phrase" : {"ip" : "10.10.10.10"}}]}}, "from" : 5, "size" : 10}`},
		{`SELECT * WHERE size>100 ORDER BY name LIMIT 0,1`, `{"query" : {"bool" : {"must" : [{"range" : {"size" : {"gt" : 100}}}]}}, "from" : 0, "size" : 1, "sort" : [{"name": "asc"}]}`},
		{`SELECT * WHERE size=10 ORDER BY name DESC LIMIT 0,1`, `{"query" : {"bool" : {"must" : [{"match_phrase" : {"size" : 10}}]}}, "from" : 0, "size" : 1, "sort" : [{"name": "desc"}]}`},
		{`SELECT * WHERE size>=100`, `{"query" : {"bool" : {"must" : [{"range" : {"size" : {"from" : 100}}}]}}}`},
		{`SELECT * WHERE name LIKE 's%'`, `{"query" : {"bool" : {"must" : [{"query_string": { "default_field": "name", "query": "s*" }}]}}}`},
		{`SELECT * WHERE name NOT LIKE 's%'`, `{"query" : {"bool" : {"must" : [{"bool" : {"must_not" : {"query_string": { "default_field": "name", "query": "s*" }}}}]}}}`},
		{`SELECT * WHERE size BETWEEN 100 AND 300`, `{"query" : {"bool" : {"must" : [{"range" : {"size" : {"from" : 100, "to" : 300}}}]}}}`},
		{`SELECT * WHERE size IN (100,300)`, `{"query" : {"bool" : {"must" : [{"terms" : {"size" : [100, 300]}}]}}}`},
		{`SELECT * WHERE size NOT IN (100,300)`, `{"query" : {"bool" : {"must" : [{"bool" : {"must_not" : {"terms" : {"size" : [100, 300]}}}}]}}}`},
		{`select * where name='sarah' and age!=40 and (country='LV' or country='AU') limit 0,1`, `{"query" : {"bool" : {"must" : [{"match_phrase" : {"name" : "sarah"}}, {"bool" : {"must_not" : [{"match_phrase" : {"age" : 40}}]}}, {"bool" : {"should" : [{"match_phrase" : {"country" : "LV"}}, {"match_phrase" : {"country" : "AU"}}]}}]}}, "from" : 0, "size" : 1}`},
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
			t.Errorf("Invalid conversion of '%s': %s, expected: %s", table.sql, result, table.converted)
		}
	}
}
