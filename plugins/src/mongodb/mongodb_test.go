package main

import (
	"fmt"
	"testing"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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
		{`SELECT * WHERE ip='10.10.10.10'`, `bson.M{"ip":"10.10.10.10"}`, `nil`, 0, 0},
		{`SELECT * WHERE ip='10.10.10.10' LIMIT 5,10`, `bson.M{"ip":"10.10.10.10"}`, `nil`, 5, 10},
		{`SELECT * WHERE size>100 ORDER BY name LIMIT 0,1`, `bson.M{"size":bson.M{"$gt":100}}`, `bson.D{bson.E{Key:"name", Value:1}}`, 0, 1},
		{`SELECT * WHERE size=10 ORDER BY name DESC LIMIT 0,1`, `bson.M{"size":10}`, `bson.D{bson.E{Key:"name", Value:-1}}`, 0, 1},
		{`SELECT * WHERE size>=100`, `bson.M{"size":bson.M{"$gte":100}}`, `nil`, 0, 1},
		{`SELECT * WHERE name LIKE 's%'`, `bson.M{"name":bson.M{"$regex":bson.Regex{Pattern:"s.*", Options:"i"}}}`, `nil`, 0, 1},
		{`SELECT * WHERE name NOT LIKE 's%'`, `bson.M{"name":bson.M{"$not":bson.M{"$regex":bson.Regex{Pattern:"s.*", Options:"i"}}}}`, `nil`, 0, 1},
		{`SELECT * WHERE size BETWEEN 100 AND 300`, `bson.M{"size":bson.M{"$gte":100, "$lte":300}}`, `nil`, 0, 1},
		{`SELECT * WHERE size NOT BETWEEN 1 AND 10`, `bson.M{"size":bson.M{"$gt":10, "$lt":1}}`, `nil`, 0, 1},
		{`SELECT * WHERE size IN (100,300)`, `bson.M{"size":bson.M{"$in":bson.A{100, 300}}}`, `nil`, 0, 1},
		{`SELECT * WHERE size NOT IN (100,300)`, `bson.M{"size":bson.M{"$nin":bson.A{100, 300}}}`, `nil`, 0, 1},
		{`select * where name='sarah' and age!=40 and (country='LV' or country='AU') limit 1`, `bson.M{"$and":bson.A{bson.M{"$and":bson.A{bson.M{"name":"sarah"}, bson.M{"age":bson.M{"$ne":40}}}}, bson.M{"$or":bson.A{bson.M{"country":"LV"}, bson.M{"country":"AU"}}}}}`, `nil`, 0, 1},
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
		filter, opts, err := c.convert(stmt, nil)
		if err != nil {
			t.Errorf("Can't convert '%s': %s", table.sql, err.Error())
			continue
		}

		if fmt.Sprintf("%#v", filter) != table.filter {
			t.Errorf("Invalid converted filter of '%s': %#v, expected: %s", table.sql, filter, table.filter)
		}

		findOpts := &options.FindOptions{}
		for _, fn := range opts.List() {
			_ = fn(findOpts)
		}

		if findOpts.Sort != nil && fmt.Sprintf("%#v", findOpts.Sort) != table.sort {
			t.Errorf("Invalid converted order of '%s': %#v, expected: %s", table.sql, findOpts.Sort, table.sort)
		}

		if findOpts.Limit != nil {
			if *findOpts.Limit != table.limit || *findOpts.Skip != table.skip {
				t.Errorf("Invalid converted options of '%s': '%v,%v', expected: '%v,%v'", table.sql, *findOpts.Skip, *findOpts.Limit, table.skip, table.limit)
			}
		}
	}
}
