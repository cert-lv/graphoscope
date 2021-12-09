/*
 * SQL to MongoDB query converter
 */

package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// convert SQL statement to the MongoDB filter & options
func (p *plugin) convert(sel *sqlparser.Select) (bson.M, *options.FindOptions, error) {

	// Handle WHERE.
	// Top level node pass in an empty interface
	// to tell the children this is root.
	// Is there any better way?
	var rootParent sqlparser.Expr

	// Values to search for
	filter, err := handleSelectWhere(&sel.Where.Expr, true, &rootParent)
	if err != nil {
		return nil, nil, err
	}

	// Handle group by
	if len(sel.GroupBy) > 0 || checkNeedAgg(sel.SelectExprs) {
		return nil, nil, errors.New("'GROUP BY' & aggregation are not supported")
	}

	// Pass these options to the Find method
	options := options.Find()

	// Offset & Rowcount validation is done by the API core
	if sel.Limit != nil {
		// Handle offset
		if sel.Limit.Offset != nil {
			offset := sqlparser.String(sel.Limit.Offset)
			queryFrom, _ := strconv.ParseInt(offset, 10, 64)
			options.SetSkip(queryFrom)
		} else {
			options.SetSkip(0)
		}

		// Handle limit
		rowcount := sqlparser.String(sel.Limit.Rowcount)
		querySize, _ := strconv.ParseInt(rowcount, 10, 64)
		options.SetLimit(querySize)
	}

	// Handle order by
	var orderByArr bson.D
	for _, orderByExpr := range sel.OrderBy {
		direction := 1
		if orderByExpr.Direction == "desc" {
			direction = -1
		}

		orderByArr = append(orderByArr, bson.E{Key: strings.Replace(sqlparser.String(orderByExpr.Expr), "`", "", -1), Value: direction})
	}

	if len(orderByArr) > 0 {
		options.SetSort(orderByArr)
	}

	return filter, options, nil
}
