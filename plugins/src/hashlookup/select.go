package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

// If the where is empty, need to check whether to agg or not
func checkNeedAgg(sqlSelect sqlparser.SelectExprs) bool {
	for _, v := range sqlSelect {
		expr, ok := v.(*sqlparser.AliasedExpr)
		if !ok {
			// No need to handle, star expression * just skip is ok
			continue
		}

		// TODO: more precise
		if _, ok := expr.Expr.(*sqlparser.FuncExpr); ok {
			return true
		}
	}

	return false
}

func handleSelectWhereComparisonExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([][2]string, error) {
	comparisonExpr := (*expr).(*sqlparser.ComparisonExpr)
	colName, ok := comparisonExpr.Left.(*sqlparser.ColName)

	if !ok {
		return nil, errors.New("Invalid comparison expression, the left must be a column name")
	}

	colNameStr := sqlparser.String(colName)
	colNameStr = strings.Replace(colNameStr, "`", "", -1)
	rightIntf, err := buildComparisonExprRightStr(comparisonExpr.Right)
	if err != nil {
		return nil, err
	}

	fields := [][2]string{}

	switch comparisonExpr.Operator {
	case "=":
		fields = append(fields, [2]string{colNameStr, fmt.Sprintf("%s", rightIntf)})
	default:
		return nil, errors.New("'=' operator is supported only")
	}

	return fields, nil
}

func handleSelectWhereAndExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([][2]string, error) {
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

	fields := append(leftStr, rightStr...)
	if topLevel {
		fields = append(fields, [2]string{"operator", "and"})
	}

	return fields, nil
}

func handleSelectWhereOrExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([][2]string, error) {
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

	fields := append(leftStr, rightStr...)
	if topLevel {
		fields = append(fields, [2]string{"operator", "or"})
	}

	return fields, nil
}

// Between a and b.
// The meaning is equal to range query
func handleSelectWhereBetweenExpr(expr *sqlparser.Expr, topLevel bool) ([][2]string, error) {
	rangeCond := (*expr).(*sqlparser.RangeCond)
	colName, ok := rangeCond.Left.(*sqlparser.ColName)

	if !ok {
		return nil, errors.New("Range column name missing")
	}

	//colNameStr := sqlparser.String(colName)
	colNameStr := strings.Trim(sqlparser.String(colName), "`")
	//fromStr := strings.Trim(sqlparser.String(rangeCond.From), "'")
	//toStr := strings.Trim(sqlparser.String(rangeCond.To), "'")

	var from string
	var to string

	// Prepare a 'From' value
	switch expr := rangeCond.From.(type) {
	case *sqlparser.SQLVal:
		switch expr.Type {
		case sqlparser.IntVal, sqlparser.FloatVal, sqlparser.StrVal:
			from = string(expr.Val)
		default:
			return nil, fmt.Errorf("Invalid BETWEEN 'from' value: %v (type %v)", string(expr.Val), expr.Type)
		}
	default:
		return nil, fmt.Errorf("Invalid BETWEEN 'from' value: %v", strings.Trim(sqlparser.String(rangeCond.From), "'"))
	}

	// Prepare a 'To' value
	switch expr := rangeCond.To.(type) {
	case *sqlparser.SQLVal:
		switch expr.Type {
		case sqlparser.IntVal, sqlparser.FloatVal, sqlparser.StrVal:
			to = string(expr.Val)
		default:
			return nil, fmt.Errorf("Invalid BETWEEN 'to' value: %v (type %v)", string(expr.Val), expr.Type)
		}
	default:
		return nil, fmt.Errorf("Invalid BETWEEN 'to' value: %v", strings.Trim(sqlparser.String(rangeCond.To), "'"))
	}

	// Build resulting fields list
	var fields [][2]string
	fields = append(fields, [2]string{colNameStr + "_from", from})
	fields = append(fields, [2]string{colNameStr + "_to", to})

	return fields, nil
}

func handleSelectWhereParenExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([][2]string, error) {
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

func buildComparisonExprRightStr(expr sqlparser.Expr) (interface{}, error) {
	var rightStr string
	var err error

	switch expr := expr.(type) {
	case *sqlparser.SQLVal:
		// Use string value type only
		rightStr = sqlparser.String(expr)
		rightStr = strings.Trim(rightStr, "'")

	case *sqlparser.BoolVal, sqlparser.BoolVal:
		rightStr = sqlparser.String(expr)

	case *sqlparser.GroupConcatExpr:
		return nil, errors.New("group_concat not supported")

	case *sqlparser.FuncExpr:
		// Parse nested
		//funcExpr := expr.(*sqlparser.FuncExpr)
		//rightStr, err = buildNestedFuncStrValue(funcExpr)
		rightStr, err = buildNestedFuncStrValue(expr)
		if err != nil {
			return nil, err
		}

	case *sqlparser.ColName:
		if sqlparser.String(expr) == "exist" {
			return nil, errors.New("'exist' expression currently not supported")
		}
		return nil, errors.New("Column name on the right side of compare operator is not supported")

	case sqlparser.ValTuple:
		rightStr = sqlparser.String(expr)

	default:
		return nil, fmt.Errorf("Unexpected SQL expression right part's type: %T", expr)
	}

	return rightStr, err
}

func handleSelectWhere(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([][2]string, error) {
	if expr == nil {
		return nil, errors.New("SQL expression cannot be nil here")
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
