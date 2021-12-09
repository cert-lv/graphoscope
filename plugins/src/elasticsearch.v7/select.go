package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

// if the where is empty, need to check whether to agg or not
func checkNeedAgg(sqlSelect sqlparser.SelectExprs) bool {
	for _, v := range sqlSelect {
		expr, ok := v.(*sqlparser.AliasedExpr)
		if !ok {
			// No need to handle, star expression * just skip is ok
			continue
		}

		//TODO more precise
		if _, ok := expr.Expr.(*sqlparser.FuncExpr); ok {
			return true
		}
	}

	return false
}

func handleSelectWhereComparisonExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (string, error) {
	comparisonExpr := (*expr).(*sqlparser.ComparisonExpr)
	colName, ok := comparisonExpr.Left.(*sqlparser.ColName)

	if !ok {
		return "", errors.New("Invalid comparison expression, the left must be a column name")
	}

	colNameStr := sqlparser.String(colName)
	colNameStr = strings.Replace(colNameStr, "`", "", -1)
	rightIntf, existsCheck, err := buildComparisonExprRightStr(comparisonExpr.Right)
	if err != nil {
		return "", err
	}

	resultStr := ""

	switch comparisonExpr.Operator {
	case "=":
		// field exists
		if existsCheck {
			resultStr = fmt.Sprintf(`{"exists":{"field":"%v"}}`, colNameStr)
		} else {
			resultStr = fmt.Sprintf(`{"match_phrase" : {"%v" : %#v}}`, colNameStr, rightIntf)
		}
	case "!=", "<>":
		if existsCheck {
			resultStr = fmt.Sprintf(`{"bool" : {"must_not" : [{"exists":{"field":"%v"}}]}}`, colNameStr)
		} else {
			resultStr = fmt.Sprintf(`{"bool" : {"must_not" : [{"match_phrase" : {"%v" : %#v}}]}}`, colNameStr, rightIntf)
		}

	case ">":
		resultStr = fmt.Sprintf(`{"range" : {"%v" : {"gt" : %#v}}}`, colNameStr, rightIntf)
	case "<":
		resultStr = fmt.Sprintf(`{"range" : {"%v" : {"lt" : %#v}}}`, colNameStr, rightIntf)
	case ">=":
		resultStr = fmt.Sprintf(`{"range" : {"%v" : {"from" : %#v}}}`, colNameStr, rightIntf)
	case "<=":
		resultStr = fmt.Sprintf(`{"range" : {"%v" : {"to" : %#v}}}`, colNameStr, rightIntf)

	case "in":
		// The default valTuple is ('1', '2', '3') like
		// so need to drop the () and replace ' to "
		rightStr := rightIntf.(string)
		rightStr = strings.Replace(rightStr, `'`, `"`, -1)
		rightStr = strings.Trim(rightStr, "(")
		rightStr = strings.Trim(rightStr, ")")

		resultStr = fmt.Sprintf(`{"terms" : {"%v" : [%v]}}`, colNameStr, rightStr)
	case "not in":
		// The default valTuple is ('1', '2', '3') like
		// so need to drop the () and replace ' to "
		rightStr := rightIntf.(string)
		rightStr = strings.Replace(rightStr, `'`, `"`, -1)
		rightStr = strings.Trim(rightStr, "(")
		rightStr = strings.Trim(rightStr, ")")

		resultStr = fmt.Sprintf(`{"bool" : {"must_not" : {"terms" : {"%v" : [%v]}}}}`, colNameStr, rightStr)

	case "like":
		rightStr := strings.Replace(rightIntf.(string), `%`, `*`, -1)
		resultStr = fmt.Sprintf(`{"query_string": { "default_field": "%v", "query": "%v" }}`, colNameStr, rightStr)
		//resultStr = fmt.Sprintf(`{"match_phrase" : {"%v" : "%v"}}`, colNameStr, rightStr)
	case "not like":
		rightStr := strings.Replace(rightIntf.(string), `%`, `*`, -1)
		resultStr = fmt.Sprintf(`{"bool" : {"must_not" : {"query_string": { "default_field": "%v", "query": "%v" }}}}`, colNameStr, rightStr)
		//resultStr = fmt.Sprintf(`{"bool" : {"must_not" : {"match_phrase" : {"%v" : "%v"}}}}`, colNameStr, rightStr)
	}

	// The root node need to have bool and must
	if topLevel {
		resultStr = fmt.Sprintf(`{"bool" : {"must" : [%v]}}`, resultStr)
	}

	return resultStr, nil
}

func handleSelectWhereAndExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (string, error) {
	andExpr := (*expr).(*sqlparser.AndExpr)
	leftExpr := andExpr.Left
	rightExpr := andExpr.Right

	leftStr, err := handleSelectWhere(&leftExpr, false, expr)
	if err != nil {
		return "", err
	}
	rightStr, err := handleSelectWhere(&rightExpr, false, expr)
	if err != nil {
		return "", err
	}

	// Not toplevel
	// if the parent node is also and, then the result can be merged

	var resultStr string
	if leftStr == "" || rightStr == "" {
		resultStr = leftStr + rightStr
	} else {
		resultStr = leftStr + ", " + rightStr
	}

	if _, ok := (*parent).(*sqlparser.AndExpr); ok {
		return resultStr, nil
	}

	return fmt.Sprintf(`{"bool" : {"must" : [%v]}}`, resultStr), nil
}

func handleSelectWhereOrExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (string, error) {
	orExpr := (*expr).(*sqlparser.OrExpr)
	leftExpr := orExpr.Left
	rightExpr := orExpr.Right

	leftStr, err := handleSelectWhere(&leftExpr, false, expr)
	if err != nil {
		return "", err
	}

	rightStr, err := handleSelectWhere(&rightExpr, false, expr)
	if err != nil {
		return "", err
	}

	var resultStr string
	if leftStr == "" || rightStr == "" {
		resultStr = leftStr + rightStr
	} else {
		resultStr = leftStr + ", " + rightStr
	}

	// Not toplevel
	// if the parent node is also or node, then merge the query param
	if _, ok := (*parent).(*sqlparser.OrExpr); ok {
		return resultStr, nil
	}

	return fmt.Sprintf(`{"bool" : {"should" : [%v]}}`, resultStr), nil
}

// between a and b.
// the meaning is equal to range query
func handleSelectWhereBetweenExpr(expr *sqlparser.Expr, topLevel bool) (string, error) {
	rangeCond := (*expr).(*sqlparser.RangeCond)
	colName, ok := rangeCond.Left.(*sqlparser.ColName)

	if !ok {
		return "", errors.New("Range column name missing")
	}

	//colNameStr := sqlparser.String(colName)
	colNameStr := strings.Trim(sqlparser.String(colName), "`")
	//fromStr := strings.Trim(sqlparser.String(rangeCond.From), "'")
	//toStr := strings.Trim(sqlparser.String(rangeCond.To), "'")

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
			return "", fmt.Errorf("Invalid BETWEEN 'from' value: %v (type %v)", string(expr.Val), expr.Type)
		}
	default:
		return "", fmt.Errorf("Invalid BETWEEN 'from' value: %v", strings.Trim(sqlparser.String(rangeCond.From), "'"))
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
			return "", fmt.Errorf("Invalid BETWEEN 'to' value: %v (type %v)", string(expr.Val), expr.Type)
		}
	default:
		return "", fmt.Errorf("Invalid BETWEEN 'to' value: %v", strings.Trim(sqlparser.String(rangeCond.To), "'"))
	}

	// Build resulting query
	resultStr := fmt.Sprintf(`{"range" : {"%v" : {"from" : %#v, "to" : %#v}}}`, colNameStr, fromIntf, toIntf)
	if topLevel {
		resultStr = fmt.Sprintf(`{"bool" : {"must" : [%v]}}`, resultStr)
	}

	return resultStr, nil
}

func handleSelectWhereParenExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (string, error) {
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
			return nil, existsCheck, fmt.Errorf("Unexpected field value's type: %v (%v)", string(expr.Val), expr.Type)
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

	return rightStr, existsCheck, nil
}

func handleSelectWhere(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) (string, error) {
	if expr == nil {
		return "", errors.New("SQL expression cannot be nil here")
	}

	switch (*expr).(type) {
	case *sqlparser.ComparisonExpr:
		return handleSelectWhereComparisonExpr(expr, topLevel, parent)

	case *sqlparser.AndExpr:
		return handleSelectWhereAndExpr(expr, topLevel, parent)

	case *sqlparser.OrExpr:
		return handleSelectWhereOrExpr(expr, topLevel, parent)

	case *sqlparser.IsExpr:
		return "", errors.New("'is' expression currently not supported")

	case *sqlparser.NotExpr:
		return "", errors.New("'not' expression currently not supported")

	case *sqlparser.RangeCond:
		return handleSelectWhereBetweenExpr(expr, topLevel)

	case *sqlparser.ParenExpr:
		return handleSelectWhereParenExpr(expr, topLevel, parent)
	}

	return "", fmt.Errorf("Unexpected SQL expression type received: %T", *expr)
}
