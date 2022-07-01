/*
 * SQL to Elasticsearch query convertor
 * Based on: https://github.com/cch123/elasticsql
 */

package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Convert SQL query to the Elasticsearch JSON query
 */
func (p *plugin) convert(sel *sqlparser.Select, fields []string) (string, error) {

	// Handle WHERE.
	// Top level node pass in an empty interface
	// to tell the children this is root.
	// Is there any better way?
	var rootParent sqlparser.Expr

	queryMapStr, err := handleSelectWhere(&sel.Where.Expr, true, &rootParent)
	if err != nil {
		return "", err
	}

	// Handle GROUP BY
	if len(sel.GroupBy) > 0 || checkNeedAgg(sel.SelectExprs) {
		return "", errors.New("'GROUP BY' & aggregation are not supported")
	}

	resultMap := make(map[string]interface{})
	resultMap["query"] = queryMapStr

	// Handle ORDER BY
	orderByArr := []string{}
	for _, orderByExpr := range sel.OrderBy {
		orderByStr := fmt.Sprintf(`{"%v": "%v"}`, strings.Replace(sqlparser.String(orderByExpr.Expr), "`", "", -1), orderByExpr.Direction)
		orderByArr = append(orderByArr, orderByStr)
	}

	if len(orderByArr) > 0 {
		resultMap["sort"] = fmt.Sprintf("[%v]", strings.Join(orderByArr, ", "))
	}

	// Handle LIMIT
	if sel.Limit != nil {
		resultMap["from"] = sqlparser.String(sel.Limit.Offset)
		resultMap["size"] = sqlparser.String(sel.Limit.Rowcount)
	}

	// Fields of the JSON to return.
	// Keep the traversal in order, avoid unpredicted JSON
	keySlice := []string{"query", "from", "size", "sort", "fields", "_source"}

	if len(fields) != 0 {
		resultMap["fields"] = "[\"" + strings.Join(fields, "\",\"") + "\"]"
		resultMap["_source"] = false
	}

	resultArr := []string{}
	for _, mapKey := range keySlice {
		if val, ok := resultMap[mapKey]; ok {
			resultArr = append(resultArr, fmt.Sprintf(`"%v" : %v`, mapKey, val))
		}
	}

	dsl := "{" + strings.Join(resultArr, ", ") + "}"
	//fmt.Println(dsl)

	// Return a JSON formatted query
	return dsl, nil
}
