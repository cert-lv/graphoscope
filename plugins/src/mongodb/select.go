package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// if the where is empty, need to check whether to agg or not
func checkNeedAgg(sqlSelect sqlparser.SelectExprs) bool {
	for _, v := range sqlSelect {
		expr, ok := v.(*sqlparser.AliasedExpr)
		if !ok {
			// No need to handle, star expression * just skip is ok
			continue
		}

		// TODO more precise
		if _, ok := expr.Expr.(*sqlparser.FuncExpr); ok {
			return true
		}
	}

	return false
}

func handleSelectWhereComparisonExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (bson.M, error) {
	comparisonExpr := (*expr).(*sqlparser.ComparisonExpr)
	colName, ok := comparisonExpr.Left.(*sqlparser.ColName)

	if !ok {
		return nil, errors.New("Invalid comparison expression, the left must be a column name")
	}

	colNameStr := sqlparser.String(colName)
	colNameStr = strings.Replace(colNameStr, "`", "", -1)
	rightIntf, existsCheck, err := buildComparisonExprRightStr(comparisonExpr.Right)
	if err != nil {
		return nil, err
	}

	var resultStr bson.M

	switch comparisonExpr.Operator {
	case "=":
		// Field exists
		if existsCheck {
			resultStr = bson.M{colNameStr: bson.M{"$exists": true}}
		} else {
			resultStr = bson.M{colNameStr: rightIntf}
		}
	case "!=", "<>":
		if existsCheck {
			resultStr = bson.M{colNameStr: bson.M{"$exists": false}}
		} else {
			resultStr = bson.M{colNameStr: bson.M{"$ne": rightIntf}}
		}

	case ">":
		resultStr = bson.M{colNameStr: bson.M{"$gt": rightIntf}}
	case "<":
		resultStr = bson.M{colNameStr: bson.M{"$lt": rightIntf}}
	case ">=":
		resultStr = bson.M{colNameStr: bson.M{"$gte": rightIntf}}
	case "<=":
		resultStr = bson.M{colNameStr: bson.M{"$lte": rightIntf}}

	case "in":
		// We need to replace the () to [] and replace ' to "
		// to be able to convert ('1', '2', '3') to bson.A{'1', '2', '3'}
		rightStr := rightIntf.(string)
		rightStr = strings.Replace(rightStr, `'`, `"`, -1)
		rightStr = strings.Trim(rightStr, "(")
		rightStr = strings.Trim(rightStr, ")")

		var list bson.A
		err = json.Unmarshal([]byte("["+rightStr+"]"), &list)
		if err != nil {
			return nil, errors.New("Can't parse array of values: " + err.Error())
		}

		resultStr = bson.M{colNameStr: bson.M{"$in": list}}
	case "not in":
		// We need to replace the () to [] and replace ' to "
		// to be able to convert ('1', '2', '3') to bson.A{'1', '2', '3'}
		rightStr := rightIntf.(string)
		rightStr = strings.Replace(rightStr, `'`, `"`, -1)
		rightStr = strings.Trim(rightStr, "(")
		rightStr = strings.Trim(rightStr, ")")

		var list bson.A
		err = json.Unmarshal([]byte("["+rightStr+"]"), &list)
		if err != nil {
			return nil, errors.New("Can't parse array of values: " + err.Error())
		}

		resultStr = bson.M{colNameStr: bson.M{"$nin": list}}

	case "like":
		rightStr := rightIntf.(string)
		rightStr = strings.Replace(rightStr, `.`, `\.`, -1)
		rightStr = strings.Replace(rightStr, `%`, `.*`, -1)

		resultStr = bson.M{colNameStr: bson.M{"$regex": primitive.Regex{Pattern: rightStr, Options: "i"}}}
	case "not like":
		rightStr := rightIntf.(string)
		rightStr = strings.Replace(rightStr, `.`, `\.`, -1)
		rightStr = strings.Replace(rightStr, `%`, `.*`, -1)

		resultStr = bson.M{colNameStr: bson.M{"$not": bson.M{"$regex": primitive.Regex{Pattern: rightStr, Options: "i"}}}}
	}

	return resultStr, nil
}

func handleSelectWhereAndExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (bson.M, error) {
	andExpr := (*expr).(*sqlparser.AndExpr)
	leftExpr := andExpr.Left
	rightExpr := andExpr.Right

	leftStr, err := handleSelectWhere(&leftExpr, false, expr)
	if err != nil {
		return nil, err
	}
	rightStr, err := handleSelectWhere(&rightExpr, false, expr)
	if err != nil {
		return nil, err
	}

	// Not toplevel
	// if the parent node is also and, then the result can be merged

	var resultStr bson.A

	for k, v := range leftStr {
		resultStr = append(resultStr, bson.M{k: v})
	}

	for k, v := range rightStr {
		resultStr = append(resultStr, bson.M{k: v})
	}

	if _, ok := (*parent).(*sqlparser.AndExpr); ok {
		return bson.M{"$and": resultStr}, nil
	}

	return bson.M{"$and": resultStr}, nil
}

func handleSelectWhereOrExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (bson.M, error) {
	orExpr := (*expr).(*sqlparser.OrExpr)
	leftExpr := orExpr.Left
	rightExpr := orExpr.Right

	leftStr, err := handleSelectWhere(&leftExpr, false, expr)
	if err != nil {
		return nil, err
	}

	rightStr, err := handleSelectWhere(&rightExpr, false, expr)
	if err != nil {
		return nil, err
	}

	var resultStr bson.A

	for k, v := range leftStr {
		resultStr = append(resultStr, bson.M{k: v})
	}

	for k, v := range rightStr {
		resultStr = append(resultStr, bson.M{k: v})
	}

	// Not toplevel
	// if the parent node is also or node, then merge the query param
	if _, ok := (*parent).(*sqlparser.OrExpr); ok {
		return bson.M{"$or": resultStr}, nil
	}

	return bson.M{"$or": resultStr}, nil
}

// between a and b.
// the meaning is equal to range query
func handleSelectWhereBetweenExpr(expr *sqlparser.Expr, topLevel bool) (bson.M, error) {
	rangeCond := (*expr).(*sqlparser.RangeCond)
	colName, ok := rangeCond.Left.(*sqlparser.ColName)

	if !ok {
		return nil, errors.New("Range column name missing")
	}

	colNameStr := strings.Trim(sqlparser.String(colName), "`")

	var fromIntf interface{}
	var toIntf interface{}

	// Prepare a valid type of the 'From' value,
	// otherwise string is everywhere
	switch expr := rangeCond.From.(type) {
	case *sqlparser.SQLVal:
		switch expr.Type {
		case sqlparser.IntVal:
			byteToInt, _ := strconv.Atoi(string(expr.Val))
			fromIntf = byteToInt

		case sqlparser.FloatVal:
			byteToFloat, _ := strconv.ParseFloat(string(expr.Val), 64)
			fromIntf = byteToFloat

		case sqlparser.StrVal:
			fromIntf = string(expr.Val)

		default:
			return nil, fmt.Errorf("Invalid BETWEEN 'from' value: %v (type %v)", string(expr.Val), expr.Type)
		}
	default:
		return nil, fmt.Errorf("Invalid BETWEEN 'from' value: %v", strings.Trim(sqlparser.String(rangeCond.From), "'"))
	}

	// Prepare a valid type of the 'To' value,
	// otherwise string is everywhere
	switch expr := rangeCond.To.(type) {
	case *sqlparser.SQLVal:
		switch expr.Type {
		case sqlparser.IntVal:
			byteToInt, _ := strconv.Atoi(string(expr.Val))
			toIntf = byteToInt

		case sqlparser.FloatVal:
			byteToFloat, _ := strconv.ParseFloat(string(expr.Val), 64)
			toIntf = byteToFloat

		case sqlparser.StrVal:
			toIntf = string(expr.Val)

		default:
			return nil, fmt.Errorf("Invalid BETWEEN 'to' value: %v (type %v)", string(expr.Val), expr.Type)
		}
	default:
		return nil, fmt.Errorf("Invalid BETWEEN 'to' value: %v", strings.Trim(sqlparser.String(rangeCond.To), "'"))
	}

	// Build resulting query
	resultStr := bson.M{colNameStr: bson.M{"$gte": fromIntf, "$lte": toIntf}}

	return resultStr, nil
}

func handleSelectWhereParenExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (bson.M, error) {
	parentBoolExpr := (*expr).(*sqlparser.ParenExpr)
	boolExpr := parentBoolExpr.Expr

	// If parent is the top level, bool must is needed
	var isThisTopLevel = false
	if topLevel {
		isThisTopLevel = true
	}

	return handleSelectWhere(&boolExpr, isThisTopLevel, parent)
}

func buildNestedFuncStrValue(nestedFunc *sqlparser.FuncExpr) (string, error) {
	return "", errors.New("Unsupported function: " + nestedFunc.Name.String())
}

func buildComparisonExprRightStr(expr sqlparser.Expr) (interface{}, bool, error) {
	var rightStr interface{}
	var err error
	var existsCheck = false

	switch expr := expr.(type) {
	case *sqlparser.SQLVal:
		// Use string value type only
		//rightStr = sqlparser.String(expr)
		//rightStr = strings.Trim(rightStr, "'")

		// Use defined value type
		switch expr.Type {
		case sqlparser.IntVal:
			byteToInt, _ := strconv.Atoi(string(expr.Val))
			rightStr = byteToInt

		case sqlparser.FloatVal:
			byteToFloat, _ := strconv.ParseFloat(string(expr.Val), 64)
			rightStr = byteToFloat

		case sqlparser.StrVal:
			rightStr = string(expr.Val)

		default:
			return nil, existsCheck, fmt.Errorf("Unexpected value: %v (type %v)", string(expr.Val), expr.Type)
		}

	case *sqlparser.BoolVal, sqlparser.BoolVal:
		rightStr, err = strconv.ParseBool(sqlparser.String(expr))
		if err != nil {
			return nil, existsCheck, errors.New("Can't parse bool value: " + err.Error())
		}

	case *sqlparser.GroupConcatExpr:
		return nil, existsCheck, errors.New("group_concat not supported")

	case *sqlparser.FuncExpr:
		// Parse nested
		//funcExpr := expr.(*sqlparser.FuncExpr)
		//rightStr, err = buildNestedFuncStrValue(funcExpr)
		rightStr, err = buildNestedFuncStrValue(expr)
		if err != nil {
			return nil, existsCheck, err
		}

	case *sqlparser.ColName:
		if sqlparser.String(expr) == "exist" {
			existsCheck = true
			return nil, existsCheck, nil
		}
		return nil, existsCheck, errors.New("Column name on the right side of compare operator is not supported")

	case sqlparser.ValTuple:
		rightStr = sqlparser.String(expr)

	default:
		return nil, existsCheck, fmt.Errorf("Unexpected SQL expression right part's type: %T", expr)
	}

	return rightStr, existsCheck, err
}

func handleSelectWhere(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (bson.M, error) {
	if expr == nil {
		return nil, errors.New("SQL expression cannot be nil")
	}

	switch (*expr).(type) {
	case *sqlparser.ComparisonExpr:
		return handleSelectWhereComparisonExpr(expr, topLevel, parent)

	case *sqlparser.AndExpr:
		return handleSelectWhereAndExpr(expr, topLevel, parent)

	case *sqlparser.OrExpr:
		return handleSelectWhereOrExpr(expr, topLevel, parent)

	case *sqlparser.IsExpr:
		return nil, errors.New("'is' expression currently not supported")

	case *sqlparser.NotExpr:
		return nil, errors.New("'not' expression currently not supported")

	case *sqlparser.RangeCond:
		return handleSelectWhereBetweenExpr(expr, topLevel)

	case *sqlparser.ParenExpr:
		return handleSelectWhereParenExpr(expr, topLevel, parent)
	}

	return nil, fmt.Errorf("Unexpected SQL expression type received: %T", *expr)
}
