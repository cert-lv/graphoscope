package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Handle single "field operator value" expression.
 *
 * Receives:
 *     expr     - SQL expression to process
 */
func handleSelectWhereComparisonExpr(expr *sqlparser.Expr) ([]string, error) {
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

	field := []string{}

	switch comparisonExpr.Operator {
	case "=":
		field = []string{colNameStr, fmt.Sprintf("%s", rightIntf)}
	default:
		return nil, errors.New("'=' operator is supported only")
	}

	return field, nil
}

/*
 * Handle "expression AND expression".
 *
 * Receives:
 *     expr     - SQL expression to process
 */
func handleSelectWhereAndExpr(expr *sqlparser.Expr) ([]string, error) {
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
	return fields, nil
}

/*
 * Handle "BETWEEN a AND b".
 *
 * Receives:
 *     expr     - SQL expression to process
 */
func handleSelectWhereBetweenExpr(expr *sqlparser.Expr) ([]string, error) {
	rangeCond := (*expr).(*sqlparser.RangeCond)

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

	fields := []string{from, to}
	return fields, nil
}

/*
 * Handle top level or groups of expressions.
 *
 * Receives:
 *     expr     - SQL expression to process
 *     topLevel - whether it's a top level expression
 *     parent   - container of the expression
 */
func handleSelectWhereParenExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([]string, error) {
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

/*
 * Check the right part of the expression
 * and return its value of specific type.
 *
 * Receives SQL expression to process
 */
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

/*
 * Handle WHERE statement.
 *
 * Receives:
 *     expr     - SQL expression to process
 *     topLevel - whether it's a top level expression
 *     parent   - container of the expression
 */
func handleSelectWhere(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([]string, error) {
	if expr == nil {
		return nil, errors.New("SQL expression cannot be nil here")
	}

	switch (*expr).(type) {
	case *sqlparser.ComparisonExpr:
		return handleSelectWhereComparisonExpr(expr)

	// Needed for datetime range
	case *sqlparser.AndExpr:
		return handleSelectWhereAndExpr(expr)

	case *sqlparser.IsExpr:
		return nil, errors.New("'is' expression currently not supported")

	case *sqlparser.NotExpr:
		return nil, errors.New("'not' expression currently not supported")

	case *sqlparser.RangeCond:
		return handleSelectWhereBetweenExpr(expr)

	case *sqlparser.ParenExpr:
		return handleSelectWhereParenExpr(expr, topLevel, parent)
	}

	return nil, fmt.Errorf("Unexpected SQL expression type received: %T", *expr)
}
