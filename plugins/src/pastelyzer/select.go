package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * If the WHERE is empty, need to check whether to agg or not
 */
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

/*
 * Handle single "field operator value" expression.
 *
 * Receives:
 *     expr     - SQL expression to process
 *     topLevel - whether it's a top level expression
 *     parent   - container of the expression
 */
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

/*
 * Handle "expression AND expression".
 *
 * Receives:
 *     expr     - SQL expression to process
 *     topLevel - whether it's a top level expression
 *     parent   - container of the expression
 */
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

	// Not toplevel
	// if the parent node is also AND, then the result can be merged

	fields := append(leftStr, rightStr...)

	return fields, nil
}

/*
 * Handle "expression OR expression".
 *
 * Receives:
 *     expr     - SQL expression to process
 *     topLevel - whether it's a top level expression
 *     parent   - container of the expression
 */
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

	// Ignore 'BETWEEN' operator, which is used al least to select datetime period
	case *sqlparser.RangeCond:
		return [][2]string{}, nil
		// return nil, errors.New("Range expression currently not supported")
		// return handleSelectWhereBetweenExpr(expr, topLevel)

	case *sqlparser.ParenExpr:
		return handleSelectWhereParenExpr(expr, topLevel, parent)
	}

	return nil, fmt.Errorf("Unexpected SQL expression type received: %T", *expr)
}
