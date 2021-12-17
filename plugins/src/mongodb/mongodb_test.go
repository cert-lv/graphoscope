package main

import (
	"fmt"
	"testing"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
 * Test SQL conversion to the data source's expected format
 */
func TestConvert(t *testing.T) {

	// Empty plugin's instance to test
	c := plugin{}

	// Pairs of SQLs and the expected results
	tables := []struct {
		sql    string
		filter string
		sort   string
		skip   int64
		limit  int64
	}{
		{`SELECT * WHERE ip='10.10.10.10'`, `primitive.M{"ip":"10.10.10.10"}`, `nil`, 0, 0},
		{`SELECT * WHERE ip='10.10.10.10' LIMIT 5,10`, `primitive.M{"ip":"10.10.10.10"}`, `nil`, 5, 10},
		{`SELECT * WHERE size>100 ORDER BY name LIMIT 0,1`, `primitive.M{"size":primitive.M{"$gt":100}}`, `primitive.M{"name":1}`, 0, 1},
		{`SELECT * WHERE size=10 ORDER BY name DESC LIMIT 0,1`, `primitive.M{"size":10}`, `primitive.M{"name":-1}`, 0, 1},
		{`SELECT * WHERE size>=100`, `primitive.M{"size":primitive.M{"$gte":100}}`, `nil`, 0, 1},
		{`SELECT * WHERE name LIKE 's%'`, `primitive.M{"name":primitive.M{"$regex":primitive.Regex{Pattern:"s.*", Options:"i"}}}`, `nil`, 0, 1},
		{`SELECT * WHERE name NOT LIKE 's%'`, `primitive.M{"name":primitive.M{"$not":primitive.M{"$regex":primitive.Regex{Pattern:"s.*", Options:"i"}}}}`, `nil`, 0, 1},
		{`SELECT * WHERE size BETWEEN 100 AND 300`, `primitive.M{"size":primitive.M{"$gte":100, "$lte":300}}`, `nil`, 0, 1},
		{`SELECT * WHERE size IN (100,300)`, `primitive.M{"size":primitive.M{"$in":primitive.A{100, 300}}}`, `nil`, 0, 1},
		{`SELECT * WHERE size NOT IN (100,300)`, `primitive.M{"size":primitive.M{"$nin":primitive.A{100, 300}}}`, `nil`, 0, 1},
		{`select * where name='sarah' and age!=40 and (country='LV' or country='AU') limit 1`, `primitive.M{"$and":primitive.A{primitive.M{"$and":primitive.A{primitive.M{"name":"sarah"}, primitive.M{"age":primitive.M{"$ne":40}}}}, primitive.M{"$or":primitive.A{primitive.M{"country":"LV"}, primitive.M{"country":"AU"}}}}}`, `nil`, 0, 1},
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
		filter, options, err := c.convert(stmt)
		if err != nil {
			t.Errorf("Can't convert '%s': %s", table.sql, err.Error())
			continue
		}

		if fmt.Sprintf("%#v", filter) != table.filter {
			t.Errorf("Invalid converted filter of '%s': %#v, expected: %s", table.sql, filter, table.filter)
		}
		if options.Sort != nil && fmt.Sprintf("%#v", options.Sort.(primitive.D).Map()) != table.sort {
			t.Errorf("Invalid converted order of '%s': %#v, expected: %s", table.sql, options.Sort.(primitive.D).Map(), table.sort)
		}

		if options.Limit != nil {
			if *options.Limit != table.limit || *options.Skip != table.skip {
				t.Errorf("Invalid converted options of '%s': '%v,%v', expected: '%v,%v'", table.sql, *options.Skip, *options.Limit, table.skip, table.limit)
			}
		}
	}
}
