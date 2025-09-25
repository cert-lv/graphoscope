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
 *     topLevel - whether it's a top level expression
 *     parent   - container of the expression
 */
func handleSelectWhereComparisonExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([2]string, error) {
	comparisonExpr := (*expr).(*sqlparser.ComparisonExpr)
	colName, ok := comparisonExpr.Left.(*sqlparser.ColName)

	if !ok {
		return [2]string{}, errors.New("Invalid comparison expression, the left must be a column name")
	}

	colNameStr := sqlparser.String(colName)
	colNameStr = strings.Replace(colNameStr, "`", "", -1)
	rightIntf, err := buildComparisonExprRightStr(comparisonExpr.Right)
	if err != nil {
		return [2]string{}, err
	}

	if comparisonExpr.Operator == "=" {
		return [2]string{colNameStr, fmt.Sprintf("%s", rightIntf)}, nil
	}

	return [2]string{}, errors.New("'=' operator is supported only")
}

/*
 * Handle top level or groups of expressions.
 *
 * Receives:
 *     expr     - SQL expression to process
 *     topLevel - whether it's a top level expression
 *     parent   - container of the expression
 */
func handleSelectWhereParenExpr(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([2]string, error) {
	parentBoolExpr := (*expr).(*sqlparser.ParenExpr)
	boolExpr := parentBoolExpr.Expr

	// If parent is the top level, bool must is needed
	var isThisTopLevel = false
	if topLevel {
		isThisTopLevel = true
	}

	return handleSelectWhere(&boolExpr, isThisTopLevel, parent)
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
		return nil, errors.New("functions are not supported")

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
func handleSelectWhere(expr *sqlparser.Expr, topLevel bool, parent *sqlparser.Expr) ([2]string, error) {
	if expr == nil {
		return [2]string{}, errors.New("SQL expression cannot be nil here")
	}

	switch (*expr).(type) {
	case *sqlparser.ComparisonExpr:
		return handleSelectWhereComparisonExpr(expr, topLevel, parent)

	case *sqlparser.ParenExpr:
		return handleSelectWhereParenExpr(expr, topLevel, parent)
	}

	return [2]string{}, fmt.Errorf("Unexpected SQL expression type received: %T", *expr)
}
