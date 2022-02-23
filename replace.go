/*
 * Replace user query fields as the data source expects.
 *
 * This way is much more advanced compared to a regexp replacing:
 *   - Much harder to break SQL syntax
 *   - Built-in fighting against SQL injections
 *   - Avoid accident replacing of the given values instead column names
 *
 * Inspired by:
 *   - https://github.com/vitessio/vitess/blob/master/go/vt/sqlparser/rewriter_api.go
 *   - https://github.com/vitessio/vitess/blob/master/go/vt/sqlparser/rewriter.go
 */

package main

import (
	"reflect"
	"runtime"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Cursor describes an SQL node encountered during 'apply' function
 */
type Cursor struct {
	// Admin defined fields to replace
	replaceFields map[string]string

	parent   sqlparser.SQLNode
	node     sqlparser.SQLNode
	replacer func(newNode, parent sqlparser.SQLNode)
}

/*
 * Traverse a syntax tree recursively, starting with root,
 * and calling 'postFunc' for each node to replace
 * fields specified in 'replaceFields' map.
 *
 * Returns a possibly modified initial syntax tree.
 *
 * Only fields that refer to AST nodes are considered to be children;
 * i.e., fields of basic types (strings, []byte, etc.) are ignored.
 */
func replaceSQL(node sqlparser.SQLNode, replaceFields map[string]string) (err error) {
	parent := &struct {
		sqlparser.SQLNode
	}{node}

	defer func() {
		if r := recover(); r != nil {
			switch rt := r.(type) {
			case *runtime.TypeAssertionError:
				err = rt
			default:
				log.Error().Msgf("Can't replace SQL fields, \"%T\" error: %v", r, rt)
			}
		}

		node = parent.SQLNode
	}()

	c := &Cursor{
		replaceFields: replaceFields,
	}

	c.apply(parent, node, nil)
	return
}

/*
 * postFunc is called for each SQL node after its children are traversed (post-order)
 */
func (c *Cursor) postFunc() {

	switch n := c.node.(type) {
	case *sqlparser.ColName:
		if sqlparser.String(n) != "" {
			colName := &sqlparser.ColName{
				Name: sqlparser.NewColIdent(c.replaceField(sqlparser.String(n))),
			}

			// replace current node in the parent field with a new object.
			// The use needs to make sure to not replace the object with something
			// of the wrong type, or the visitor will panic
			c.replacer(colName, c.parent)
			n = colName
		}
	// Handle internal groups like "in", "like", () recursively
	case *sqlparser.ParenExpr:
		err := replaceSQL(n.Expr, c.replaceFields)
		if err != nil {
			log.Error().
				Str("sql", sqlparser.String(n.Expr)).
				Msg("Can't replace fields in the SQL statement: " + err.Error())
		}
	}
}

/*
 * replaceField actually does a field renaming
 */
func (c *Cursor) replaceField(field string) string {
	for k, v := range c.replaceFields {
		if field == k {
			return v
		}
	}

	return field
}

/*
 * Check whether current node is nil
 */
func (c *Cursor) isNilNode(i interface{}) bool {
	valueOf := reflect.ValueOf(i)
	kind := valueOf.Kind()
	isNullable := kind == reflect.Ptr || kind == reflect.Array || kind == reflect.Slice

	return isNullable && valueOf.IsNil()
}

/*
 * 'apply' is invoked by 'replaceSQL' for each node, even if it's nil,
 * before and/or after the node's children.
 * Commented out code is left for the possible future needs
 */
func (c *Cursor) apply(parent, node sqlparser.SQLNode, replacer func(newNode, parent sqlparser.SQLNode)) {
	if node == nil || c.isNilNode(node) {
		return
	}

	savedReplacer := c.replacer
	savedNode := c.node
	savedParent := c.parent

	c.replacer = replacer
	c.node = node
	c.parent = parent

	switch n := node.(type) {
	// case *sqlparser.AddColumns:
	// 	for x, el := range n.Columns {
	// 		c.apply(node, el, func(idx int) func(sqlparser.SQLNode, sqlparser.SQLNode) {
	// 			return func(newNode, container sqlparser.SQLNode) {
	// 				container.(*AddColumns).Columns[idx] = newNode.(*ColumnDefinition)
	// 			}
	// 		}(x))
	// 	}
	// 	c.apply(node, n.First, func(newNode, parent sqlparser.SQLNode) {
	// 		parent.(*AddColumns).First = newNode.(*ColName)
	// 	})
	// 	c.apply(node, n.After, func(newNode, parent sqlparser.SQLNode) {
	// 		parent.(*AddColumns).After = newNode.(*ColName)
	// 	})
	// case *AddConstraintDefinition:
	// 	c.apply(node, n.ConstraintDefinition, func(newNode, parent SQLNode) {
	// 		parent.(*AddConstraintDefinition).ConstraintDefinition = newNode.(*ConstraintDefinition)
	// 	})
	// case *AddIndexDefinition:
	// 	c.apply(node, n.IndexDefinition, func(newNode, parent SQLNode) {
	// 		parent.(*AddIndexDefinition).IndexDefinition = newNode.(*IndexDefinition)
	// 	})
	// case *AliasedExpr:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*AliasedExpr).Expr = newNode.(Expr)
	// 	})
	// 	c.apply(node, n.As, func(newNode, parent SQLNode) {
	// 		parent.(*AliasedExpr).As = newNode.(ColIdent)
	// 	})
	// case *AliasedTableExpr:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*AliasedTableExpr).Expr = newNode.(SimpleTableExpr)
	// 	})
	// 	c.apply(node, n.Partitions, func(newNode, parent SQLNode) {
	// 		parent.(*AliasedTableExpr).Partitions = newNode.(Partitions)
	// 	})
	// 	c.apply(node, n.As, func(newNode, parent SQLNode) {
	// 		parent.(*AliasedTableExpr).As = newNode.(TableIdent)
	// 	})
	// 	c.apply(node, n.Hints, func(newNode, parent SQLNode) {
	// 		parent.(*AliasedTableExpr).Hints = newNode.(*IndexHints)
	// 	})
	// case *AlterCharset:
	// case *AlterColumn:
	// 	c.apply(node, n.Column, func(newNode, parent SQLNode) {
	// 		parent.(*AlterColumn).Column = newNode.(*ColName)
	// 	})
	// 	c.apply(node, n.DefaultVal, func(newNode, parent SQLNode) {
	// 		parent.(*AlterColumn).DefaultVal = newNode.(Expr)
	// 	})
	// case *AlterDatabase:
	// case *AlterTable:
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*AlterTable).Table = newNode.(TableName)
	// 	})
	// 	for x, el := range n.AlterOptions {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*AlterTable).AlterOptions[idx] = newNode.(AlterOption)
	// 			}
	// 		}(x))
	// 	}
	// 	c.apply(node, n.PartitionSpec, func(newNode, parent SQLNode) {
	// 		parent.(*AlterTable).PartitionSpec = newNode.(*PartitionSpec)
	// 	})
	// case *AlterView:
	// 	c.apply(node, n.ViewName, func(newNode, parent SQLNode) {
	// 		parent.(*AlterView).ViewName = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.Columns, func(newNode, parent SQLNode) {
	// 		parent.(*AlterView).Columns = newNode.(Columns)
	// 	})
	// 	c.apply(node, n.Select, func(newNode, parent SQLNode) {
	// 		parent.(*AlterView).Select = newNode.(SelectStatement)
	// 	})
	// case *AlterVschema:
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*AlterVschema).Table = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.VindexSpec, func(newNode, parent SQLNode) {
	// 		parent.(*AlterVschema).VindexSpec = newNode.(*VindexSpec)
	// 	})
	// 	for x, el := range n.VindexCols {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*AlterVschema).VindexCols[idx] = newNode.(ColIdent)
	// 			}
	// 		}(x))
	// 	}
	// 	c.apply(node, n.AutoIncSpec, func(newNode, parent SQLNode) {
	// 		parent.(*AlterVschema).AutoIncSpec = newNode.(*AutoIncSpec)
	// 	})
	case *sqlparser.AndExpr:
		c.apply(node, n.Left, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.AndExpr).Left = newNode.(sqlparser.Expr)
		})
		c.apply(node, n.Right, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.AndExpr).Right = newNode.(sqlparser.Expr)
		})
	// case Argument:
	// case *AutoIncSpec:
	// 	c.apply(node, n.Column, func(newNode, parent SQLNode) {
	// 		parent.(*AutoIncSpec).Column = newNode.(ColIdent)
	// 	})
	// 	c.apply(node, n.Sequence, func(newNode, parent SQLNode) {
	// 		parent.(*AutoIncSpec).Sequence = newNode.(TableName)
	// 	})
	// case *Begin:
	// case *BinaryExpr:
	// 	c.apply(node, n.Left, func(newNode, parent SQLNode) {
	// 		parent.(*BinaryExpr).Left = newNode.(Expr)
	// 	})
	// 	c.apply(node, n.Right, func(newNode, parent SQLNode) {
	// 		parent.(*BinaryExpr).Right = newNode.(Expr)
	// 	})
	// case *CallProc:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*CallProc).Name = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.Params, func(newNode, parent SQLNode) {
	// 		parent.(*CallProc).Params = newNode.(Exprs)
	// 	})
	// case *CaseExpr:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*CaseExpr).Expr = newNode.(Expr)
	// 	})
	// 	for x, el := range n.Whens {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*CaseExpr).Whens[idx] = newNode.(*When)
	// 			}
	// 		}(x))
	// 	}
	// 	c.apply(node, n.Else, func(newNode, parent SQLNode) {
	// 		parent.(*CaseExpr).Else = newNode.(Expr)
	// 	})
	// case *ChangeColumn:
	// 	c.apply(node, n.OldColumn, func(newNode, parent SQLNode) {
	// 		parent.(*ChangeColumn).OldColumn = newNode.(*ColName)
	// 	})
	// 	c.apply(node, n.NewColDefinition, func(newNode, parent SQLNode) {
	// 		parent.(*ChangeColumn).NewColDefinition = newNode.(*ColumnDefinition)
	// 	})
	// 	c.apply(node, n.First, func(newNode, parent SQLNode) {
	// 		parent.(*ChangeColumn).First = newNode.(*ColName)
	// 	})
	// 	c.apply(node, n.After, func(newNode, parent SQLNode) {
	// 		parent.(*ChangeColumn).After = newNode.(*ColName)
	// 	})
	// case *CheckConstraintDefinition:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*CheckConstraintDefinition).Expr = newNode.(Expr)
	// 	})
	case sqlparser.ColIdent:
	case *sqlparser.ColName:
		c.apply(node, n.Name, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.ColName).Name = newNode.(sqlparser.ColIdent)
		})
		c.apply(node, n.Qualifier, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.ColName).Qualifier = newNode.(sqlparser.TableName)
		})
	// case *CollateExpr:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*CollateExpr).Expr = newNode.(Expr)
	// 	})
	// case *ColumnDefinition:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*ColumnDefinition).Name = newNode.(ColIdent)
	// 	})
	// case *ColumnType:
	// 	c.apply(node, n.Length, func(newNode, parent SQLNode) {
	// 		parent.(*ColumnType).Length = newNode.(*Literal)
	// 	})
	// 	c.apply(node, n.Scale, func(newNode, parent SQLNode) {
	// 		parent.(*ColumnType).Scale = newNode.(*Literal)
	// 	})
	// case Columns:
	// 	for x, el := range n {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(Columns)[idx] = newNode.(ColIdent)
	// 			}
	// 		}(x))
	// 	}
	// case Comments:
	// case *Commit:
	case *sqlparser.ComparisonExpr:
		c.apply(node, n.Left, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.ComparisonExpr).Left = newNode.(sqlparser.Expr)
		})
		c.apply(node, n.Right, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.ComparisonExpr).Right = newNode.(sqlparser.Expr)
		})
		c.apply(node, n.Escape, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.ComparisonExpr).Escape = newNode.(sqlparser.Expr)
		})
	// case *ConstraintDefinition:
	// 	c.apply(node, n.Details, func(newNode, parent SQLNode) {
	// 		parent.(*ConstraintDefinition).Details = newNode.(ConstraintInfo)
	// 	})
	// case *ConvertExpr:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*ConvertExpr).Expr = newNode.(Expr)
	// 	})
	// 	c.apply(node, n.Type, func(newNode, parent SQLNode) {
	// 		parent.(*ConvertExpr).Type = newNode.(*ConvertType)
	// 	})
	// case *ConvertType:
	// 	c.apply(node, n.Length, func(newNode, parent SQLNode) {
	// 		parent.(*ConvertType).Length = newNode.(*Literal)
	// 	})
	// 	c.apply(node, n.Scale, func(newNode, parent SQLNode) {
	// 		parent.(*ConvertType).Scale = newNode.(*Literal)
	// 	})
	// case *ConvertUsingExpr:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*ConvertUsingExpr).Expr = newNode.(Expr)
	// 	})
	// case *CreateDatabase:
	// case *CreateTable:
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*CreateTable).Table = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.TableSpec, func(newNode, parent SQLNode) {
	// 		parent.(*CreateTable).TableSpec = newNode.(*TableSpec)
	// 	})
	// 	c.apply(node, n.OptLike, func(newNode, parent SQLNode) {
	// 		parent.(*CreateTable).OptLike = newNode.(*OptLike)
	// 	})
	// case *CreateView:
	// 	c.apply(node, n.ViewName, func(newNode, parent SQLNode) {
	// 		parent.(*CreateView).ViewName = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.Columns, func(newNode, parent SQLNode) {
	// 		parent.(*CreateView).Columns = newNode.(Columns)
	// 	})
	// 	c.apply(node, n.Select, func(newNode, parent SQLNode) {
	// 		parent.(*CreateView).Select = newNode.(SelectStatement)
	// 	})
	// case *CurTimeFuncExpr:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*CurTimeFuncExpr).Name = newNode.(ColIdent)
	// 	})
	// 	c.apply(node, n.Fsp, func(newNode, parent SQLNode) {
	// 		parent.(*CurTimeFuncExpr).Fsp = newNode.(Expr)
	// 	})
	// case *Default:
	// case *Delete:
	// 	c.apply(node, n.Comments, func(newNode, parent SQLNode) {
	// 		parent.(*Delete).Comments = newNode.(Comments)
	// 	})
	// 	c.apply(node, n.Targets, func(newNode, parent SQLNode) {
	// 		parent.(*Delete).Targets = newNode.(TableNames)
	// 	})
	// 	c.apply(node, n.TableExprs, func(newNode, parent SQLNode) {
	// 		parent.(*Delete).TableExprs = newNode.(TableExprs)
	// 	})
	// 	c.apply(node, n.Partitions, func(newNode, parent SQLNode) {
	// 		parent.(*Delete).Partitions = newNode.(Partitions)
	// 	})
	// 	c.apply(node, n.Where, func(newNode, parent SQLNode) {
	// 		parent.(*Delete).Where = newNode.(*Where)
	// 	})
	// 	c.apply(node, n.OrderBy, func(newNode, parent SQLNode) {
	// 		parent.(*Delete).OrderBy = newNode.(OrderBy)
	// 	})
	// 	c.apply(node, n.Limit, func(newNode, parent SQLNode) {
	// 		parent.(*Delete).Limit = newNode.(*Limit)
	// 	})
	// case *DerivedTable:
	// 	c.apply(node, n.Select, func(newNode, parent SQLNode) {
	// 		parent.(*DerivedTable).Select = newNode.(SelectStatement)
	// 	})
	// case *DropColumn:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*DropColumn).Name = newNode.(*ColName)
	// 	})
	// case *DropDatabase:
	// case *DropKey:
	// case *DropTable:
	// 	c.apply(node, n.FromTables, func(newNode, parent SQLNode) {
	// 		parent.(*DropTable).FromTables = newNode.(TableNames)
	// 	})
	// case *DropView:
	// 	c.apply(node, n.FromTables, func(newNode, parent SQLNode) {
	// 		parent.(*DropView).FromTables = newNode.(TableNames)
	// 	})
	// case *ExistsExpr:
	// 	c.apply(node, n.Subquery, func(newNode, parent SQLNode) {
	// 		parent.(*ExistsExpr).Subquery = newNode.(*Subquery)
	// 	})
	// case *ExplainStmt:
	// 	c.apply(node, n.Statement, func(newNode, parent SQLNode) {
	// 		parent.(*ExplainStmt).Statement = newNode.(Statement)
	// 	})
	// case *ExplainTab:
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*ExplainTab).Table = newNode.(TableName)
	// 	})
	case sqlparser.Exprs:
		for x, el := range n {
			c.apply(node, el, func(idx int) func(sqlparser.SQLNode, sqlparser.SQLNode) {
				return func(newNode, container sqlparser.SQLNode) {
					container.(sqlparser.Exprs)[idx] = newNode.(sqlparser.Expr)
				}
			}(x))
		}
	// case *Flush:
	// 	c.apply(node, n.TableNames, func(newNode, parent SQLNode) {
	// 		parent.(*Flush).TableNames = newNode.(TableNames)
	// 	})
	// case *Force:
	// case *ForeignKeyDefinition:
	// 	c.apply(node, n.Source, func(newNode, parent SQLNode) {
	// 		parent.(*ForeignKeyDefinition).Source = newNode.(Columns)
	// 	})
	// 	c.apply(node, n.ReferencedTable, func(newNode, parent SQLNode) {
	// 		parent.(*ForeignKeyDefinition).ReferencedTable = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.ReferencedColumns, func(newNode, parent SQLNode) {
	// 		parent.(*ForeignKeyDefinition).ReferencedColumns = newNode.(Columns)
	// 	})
	// 	c.apply(node, n.OnDelete, func(newNode, parent SQLNode) {
	// 		parent.(*ForeignKeyDefinition).OnDelete = newNode.(ReferenceAction)
	// 	})
	// 	c.apply(node, n.OnUpdate, func(newNode, parent SQLNode) {
	// 		parent.(*ForeignKeyDefinition).OnUpdate = newNode.(ReferenceAction)
	// 	})
	case *sqlparser.FuncExpr:
		c.apply(node, n.Qualifier, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.FuncExpr).Qualifier = newNode.(sqlparser.TableIdent)
		})
		c.apply(node, n.Name, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.FuncExpr).Name = newNode.(sqlparser.ColIdent)
		})
		c.apply(node, n.Exprs, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.FuncExpr).Exprs = newNode.(sqlparser.SelectExprs)
		})
	case sqlparser.GroupBy:
		for x, el := range n {
			c.apply(node, el, func(idx int) func(sqlparser.SQLNode, sqlparser.SQLNode) {
				return func(newNode, container sqlparser.SQLNode) {
					container.(sqlparser.GroupBy)[idx] = newNode.(sqlparser.Expr)
				}
			}(x))
		}
	// case *GroupConcatExpr:
	// 	c.apply(node, n.Exprs, func(newNode, parent SQLNode) {
	// 		parent.(*GroupConcatExpr).Exprs = newNode.(SelectExprs)
	// 	})
	// 	c.apply(node, n.OrderBy, func(newNode, parent SQLNode) {
	// 		parent.(*GroupConcatExpr).OrderBy = newNode.(OrderBy)
	// 	})
	// 	c.apply(node, n.Limit, func(newNode, parent SQLNode) {
	// 		parent.(*GroupConcatExpr).Limit = newNode.(*Limit)
	// 	})
	// case *IndexDefinition:
	// 	c.apply(node, n.Info, func(newNode, parent SQLNode) {
	// 		parent.(*IndexDefinition).Info = newNode.(*IndexInfo)
	// 	})
	// case *IndexHints:
	// 	for x, el := range n.Indexes {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*IndexHints).Indexes[idx] = newNode.(ColIdent)
	// 			}
	// 		}(x))
	// 	}
	// case *IndexInfo:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*IndexInfo).Name = newNode.(ColIdent)
	// 	})
	// 	c.apply(node, n.ConstraintName, func(newNode, parent SQLNode) {
	// 		parent.(*IndexInfo).ConstraintName = newNode.(ColIdent)
	// 	})
	// case *Insert:
	// 	c.apply(node, n.Comments, func(newNode, parent SQLNode) {
	// 		parent.(*Insert).Comments = newNode.(Comments)
	// 	})
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*Insert).Table = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.Partitions, func(newNode, parent SQLNode) {
	// 		parent.(*Insert).Partitions = newNode.(Partitions)
	// 	})
	// 	c.apply(node, n.Columns, func(newNode, parent SQLNode) {
	// 		parent.(*Insert).Columns = newNode.(Columns)
	// 	})
	// 	c.apply(node, n.Rows, func(newNode, parent SQLNode) {
	// 		parent.(*Insert).Rows = newNode.(InsertRows)
	// 	})
	// 	c.apply(node, n.OnDup, func(newNode, parent SQLNode) {
	// 		parent.(*Insert).OnDup = newNode.(OnDup)
	// 	})
	// case *IntervalExpr:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*IntervalExpr).Expr = newNode.(Expr)
	// 	})
	case *sqlparser.IsExpr:
		c.apply(node, n.Expr, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.IsExpr).Expr = newNode.(sqlparser.Expr)
		})
	// case JoinCondition:
	// 	c.apply(node, n.On, replacePanic("JoinCondition On"))
	// 	c.apply(node, n.Using, replacePanic("JoinCondition Using"))
	// case *JoinTableExpr:
	// 	c.apply(node, n.LeftExpr, func(newNode, parent SQLNode) {
	// 		parent.(*JoinTableExpr).LeftExpr = newNode.(TableExpr)
	// 	})
	// 	c.apply(node, n.RightExpr, func(newNode, parent SQLNode) {
	// 		parent.(*JoinTableExpr).RightExpr = newNode.(TableExpr)
	// 	})
	// 	c.apply(node, n.Condition, func(newNode, parent SQLNode) {
	// 		parent.(*JoinTableExpr).Condition = newNode.(JoinCondition)
	// 	})
	//case *sqlparser.KeyState:
	case *sqlparser.Limit:
		c.apply(node, n.Offset, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Limit).Offset = newNode.(sqlparser.Expr)
		})
		c.apply(node, n.Rowcount, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Limit).Rowcount = newNode.(sqlparser.Expr)
		})
	// case ListArg:
	// case *Literal:
	// case *Load:
	// case *LockOption:
	// case *LockTables:
	case *sqlparser.MatchExpr:
		c.apply(node, n.Columns, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.MatchExpr).Columns = newNode.(sqlparser.SelectExprs)
		})
		c.apply(node, n.Expr, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.MatchExpr).Expr = newNode.(sqlparser.Expr)
		})
	// case *ModifyColumn:
	// 	c.apply(node, n.NewColDefinition, func(newNode, parent SQLNode) {
	// 		parent.(*ModifyColumn).NewColDefinition = newNode.(*ColumnDefinition)
	// 	})
	// 	c.apply(node, n.First, func(newNode, parent SQLNode) {
	// 		parent.(*ModifyColumn).First = newNode.(*ColName)
	// 	})
	// 	c.apply(node, n.After, func(newNode, parent SQLNode) {
	// 		parent.(*ModifyColumn).After = newNode.(*ColName)
	// 	})
	// case *Nextval:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*Nextval).Expr = newNode.(Expr)
	// 	})
	case *sqlparser.NotExpr:
		c.apply(node, n.Expr, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.NotExpr).Expr = newNode.(sqlparser.Expr)
		})
	case *sqlparser.NullVal:
	case sqlparser.OnDup:
		for x, el := range n {
			c.apply(node, el, func(idx int) func(sqlparser.SQLNode, sqlparser.SQLNode) {
				return func(newNode, container sqlparser.SQLNode) {
					container.(sqlparser.OnDup)[idx] = newNode.(*sqlparser.UpdateExpr)
				}
			}(x))
		}
	// case *OptLike:
	// 	c.apply(node, n.LikeTable, func(newNode, parent SQLNode) {
	// 		parent.(*OptLike).LikeTable = newNode.(TableName)
	// 	})
	case *sqlparser.OrExpr:
		c.apply(node, n.Left, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.OrExpr).Left = newNode.(sqlparser.Expr)
		})
		c.apply(node, n.Right, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.OrExpr).Right = newNode.(sqlparser.Expr)
		})
	case *sqlparser.Order:
		c.apply(node, n.Expr, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Order).Expr = newNode.(sqlparser.Expr)
		})
	case sqlparser.OrderBy:
		for x, el := range n {
			c.apply(node, el, func(idx int) func(sqlparser.SQLNode, sqlparser.SQLNode) {
				return func(newNode, container sqlparser.SQLNode) {
					container.(sqlparser.OrderBy)[idx] = newNode.(*sqlparser.Order)
				}
			}(x))
		}
	// case *OrderByOption:
	// 	c.apply(node, n.Cols, func(newNode, parent SQLNode) {
	// 		parent.(*OrderByOption).Cols = newNode.(Columns)
	// 	})
	// case *OtherAdmin:
	// case *OtherRead:
	case *sqlparser.ParenSelect:
		c.apply(node, n.Select, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.ParenSelect).Select = newNode.(sqlparser.SelectStatement)
		})
	case *sqlparser.ParenTableExpr:
		c.apply(node, n.Exprs, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.ParenTableExpr).Exprs = newNode.(sqlparser.TableExprs)
		})
	// case *PartitionDefinition:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*PartitionDefinition).Name = newNode.(ColIdent)
	// 	})
	// 	c.apply(node, n.Limit, func(newNode, parent SQLNode) {
	// 		parent.(*PartitionDefinition).Limit = newNode.(Expr)
	// 	})
	// case *PartitionSpec:
	// 	c.apply(node, n.Names, func(newNode, parent SQLNode) {
	// 		parent.(*PartitionSpec).Names = newNode.(Partitions)
	// 	})
	// 	c.apply(node, n.Number, func(newNode, parent SQLNode) {
	// 		parent.(*PartitionSpec).Number = newNode.(*Literal)
	// 	})
	// 	c.apply(node, n.TableName, func(newNode, parent SQLNode) {
	// 		parent.(*PartitionSpec).TableName = newNode.(TableName)
	// 	})
	// 	for x, el := range n.Definitions {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*PartitionSpec).Definitions[idx] = newNode.(*PartitionDefinition)
	// 			}
	// 		}(x))
	// 	}
	// case Partitions:
	// 	for x, el := range n {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(Partitions)[idx] = newNode.(ColIdent)
	// 			}
	// 		}(x))
	// 	}
	case *sqlparser.RangeCond:
		c.apply(node, n.Left, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.RangeCond).Left = newNode.(sqlparser.Expr)
		})
		c.apply(node, n.From, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.RangeCond).From = newNode.(sqlparser.Expr)
		})
		c.apply(node, n.To, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.RangeCond).To = newNode.(sqlparser.Expr)
		})
	// case *Release:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*Release).Name = newNode.(ColIdent)
	// 	})
	// case *RenameIndex:
	// case *RenameTable:
	// case *RenameTableName:
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*RenameTableName).Table = newNode.(TableName)
	// 	})
	// case *Rollback:
	// case *SRollback:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*SRollback).Name = newNode.(ColIdent)
	// 	})
	// case *Savepoint:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*Savepoint).Name = newNode.(ColIdent)
	// 	})
	case *sqlparser.Select:
		c.apply(node, n.Comments, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Select).Comments = newNode.(sqlparser.Comments)
		})
		c.apply(node, n.SelectExprs, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Select).SelectExprs = newNode.(sqlparser.SelectExprs)
		})
		c.apply(node, n.From, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Select).From = newNode.(sqlparser.TableExprs)
		})
		c.apply(node, n.Where, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Select).Where = newNode.(*sqlparser.Where)
		})
		c.apply(node, n.GroupBy, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Select).GroupBy = newNode.(sqlparser.GroupBy)
		})
		c.apply(node, n.Having, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Select).Having = newNode.(*sqlparser.Where)
		})
		c.apply(node, n.OrderBy, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Select).OrderBy = newNode.(sqlparser.OrderBy)
		})
		c.apply(node, n.Limit, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Select).Limit = newNode.(*sqlparser.Limit)
		})
		// c.apply(node, n.Into, func(newNode, parent SQLNode) {
		// 	parent.(*Select).Into = newNode.(*SelectInto)
		// })
	case sqlparser.SelectExprs:
		for x, el := range n {
			c.apply(node, el, func(idx int) func(sqlparser.SQLNode, sqlparser.SQLNode) {
				return func(newNode, container sqlparser.SQLNode) {
					container.(sqlparser.SelectExprs)[idx] = newNode.(sqlparser.SelectExpr)
				}
			}(x))
		}
	// case *SelectInto:
	// case *Set:
	// 	c.apply(node, n.Comments, func(newNode, parent SQLNode) {
	// 		parent.(*Set).Comments = newNode.(Comments)
	// 	})
	// 	c.apply(node, n.Exprs, func(newNode, parent SQLNode) {
	// 		parent.(*Set).Exprs = newNode.(SetExprs)
	// 	})
	// case *SetExpr:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*SetExpr).Name = newNode.(ColIdent)
	// 	})
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*SetExpr).Expr = newNode.(Expr)
	// 	})
	// case SetExprs:
	// 	for x, el := range n {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(SetExprs)[idx] = newNode.(*SetExpr)
	// 			}
	// 		}(x))
	// 	}
	// case *SetTransaction:
	// 	c.apply(node, n.SQLNode, func(newNode, parent SQLNode) {
	// 		parent.(*SetTransaction).SQLNode = newNode.(SQLNode)
	// 	})
	// 	c.apply(node, n.Comments, func(newNode, parent SQLNode) {
	// 		parent.(*SetTransaction).Comments = newNode.(Comments)
	// 	})
	// 	for x, el := range n.Characteristics {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*SetTransaction).Characteristics[idx] = newNode.(Characteristic)
	// 			}
	// 		}(x))
	// 	}
	// case *Show:
	// 	c.apply(node, n.Internal, func(newNode, parent SQLNode) {
	// 		parent.(*Show).Internal = newNode.(ShowInternal)
	// 	})
	// case *ShowBasic:
	// 	c.apply(node, n.Tbl, func(newNode, parent SQLNode) {
	// 		parent.(*ShowBasic).Tbl = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.Filter, func(newNode, parent SQLNode) {
	// 		parent.(*ShowBasic).Filter = newNode.(*ShowFilter)
	// 	})
	// case *ShowCreate:
	// 	c.apply(node, n.Op, func(newNode, parent SQLNode) {
	// 		parent.(*ShowCreate).Op = newNode.(TableName)
	// 	})
	// case *ShowFilter:
	// 	c.apply(node, n.Filter, func(newNode, parent SQLNode) {
	// 		parent.(*ShowFilter).Filter = newNode.(Expr)
	// 	})
	// case *ShowLegacy:
	// 	c.apply(node, n.OnTable, func(newNode, parent SQLNode) {
	// 		parent.(*ShowLegacy).OnTable = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*ShowLegacy).Table = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.ShowCollationFilterOpt, func(newNode, parent SQLNode) {
	// 		parent.(*ShowLegacy).ShowCollationFilterOpt = newNode.(Expr)
	// 	})
	// case *StarExpr:
	// 	c.apply(node, n.TableName, func(newNode, parent SQLNode) {
	// 		parent.(*StarExpr).TableName = newNode.(TableName)
	// 	})
	// case *Stream:
	// 	c.apply(node, n.Comments, func(newNode, parent SQLNode) {
	// 		parent.(*Stream).Comments = newNode.(Comments)
	// 	})
	// 	c.apply(node, n.SelectExpr, func(newNode, parent SQLNode) {
	// 		parent.(*Stream).SelectExpr = newNode.(SelectExpr)
	// 	})
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*Stream).Table = newNode.(TableName)
	// 	})
	// case *Subquery:
	// 	c.apply(node, n.Select, func(newNode, parent SQLNode) {
	// 		parent.(*Subquery).Select = newNode.(SelectStatement)
	// 	})
	// case *SubstrExpr:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*SubstrExpr).Name = newNode.(*ColName)
	// 	})
	// 	c.apply(node, n.StrVal, func(newNode, parent SQLNode) {
	// 		parent.(*SubstrExpr).StrVal = newNode.(*Literal)
	// 	})
	// 	c.apply(node, n.From, func(newNode, parent SQLNode) {
	// 		parent.(*SubstrExpr).From = newNode.(Expr)
	// 	})
	// 	c.apply(node, n.To, func(newNode, parent SQLNode) {
	// 		parent.(*SubstrExpr).To = newNode.(Expr)
	// 	})
	// case TableExprs:
	// 	for x, el := range n {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(TableExprs)[idx] = newNode.(TableExpr)
	// 			}
	// 		}(x))
	// 	}
	// case TableIdent:
	// case TableName:
	// 	c.apply(node, n.Name, replacePanic("TableName Name"))
	// 	c.apply(node, n.Qualifier, replacePanic("TableName Qualifier"))
	// case TableNames:
	// 	for x, el := range n {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(TableNames)[idx] = newNode.(TableName)
	// 			}
	// 		}(x))
	// 	}
	// case TableOptions:
	// case *TableSpec:
	// 	for x, el := range n.Columns {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*TableSpec).Columns[idx] = newNode.(*ColumnDefinition)
	// 			}
	// 		}(x))
	// 	}
	// 	for x, el := range n.Indexes {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*TableSpec).Indexes[idx] = newNode.(*IndexDefinition)
	// 			}
	// 		}(x))
	// 	}
	// 	for x, el := range n.Constraints {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*TableSpec).Constraints[idx] = newNode.(*ConstraintDefinition)
	// 			}
	// 		}(x))
	// 	}
	// 	c.apply(node, n.Options, func(newNode, parent SQLNode) {
	// 		parent.(*TableSpec).Options = newNode.(TableOptions)
	// 	})
	// case *TablespaceOperation:
	// case *TimestampFuncExpr:
	// 	c.apply(node, n.Expr1, func(newNode, parent SQLNode) {
	// 		parent.(*TimestampFuncExpr).Expr1 = newNode.(Expr)
	// 	})
	// 	c.apply(node, n.Expr2, func(newNode, parent SQLNode) {
	// 		parent.(*TimestampFuncExpr).Expr2 = newNode.(Expr)
	// 	})
	// case *TruncateTable:
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*TruncateTable).Table = newNode.(TableName)
	// 	})
	// case *UnaryExpr:
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*UnaryExpr).Expr = newNode.(Expr)
	// 	})
	// case *Union:
	// 	c.apply(node, n.FirstStatement, func(newNode, parent SQLNode) {
	// 		parent.(*Union).FirstStatement = newNode.(SelectStatement)
	// 	})
	// 	for x, el := range n.UnionSelects {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*Union).UnionSelects[idx] = newNode.(*UnionSelect)
	// 			}
	// 		}(x))
	// 	}
	// 	c.apply(node, n.OrderBy, func(newNode, parent SQLNode) {
	// 		parent.(*Union).OrderBy = newNode.(OrderBy)
	// 	})
	// 	c.apply(node, n.Limit, func(newNode, parent SQLNode) {
	// 		parent.(*Union).Limit = newNode.(*Limit)
	// 	})
	// case *UnionSelect:
	// 	c.apply(node, n.Statement, func(newNode, parent SQLNode) {
	// 		parent.(*UnionSelect).Statement = newNode.(SelectStatement)
	// 	})
	// case *UnlockTables:
	// case *Update:
	// 	c.apply(node, n.Comments, func(newNode, parent SQLNode) {
	// 		parent.(*Update).Comments = newNode.(Comments)
	// 	})
	// 	c.apply(node, n.TableExprs, func(newNode, parent SQLNode) {
	// 		parent.(*Update).TableExprs = newNode.(TableExprs)
	// 	})
	// 	c.apply(node, n.Exprs, func(newNode, parent SQLNode) {
	// 		parent.(*Update).Exprs = newNode.(UpdateExprs)
	// 	})
	// 	c.apply(node, n.Where, func(newNode, parent SQLNode) {
	// 		parent.(*Update).Where = newNode.(*Where)
	// 	})
	// 	c.apply(node, n.OrderBy, func(newNode, parent SQLNode) {
	// 		parent.(*Update).OrderBy = newNode.(OrderBy)
	// 	})
	// 	c.apply(node, n.Limit, func(newNode, parent SQLNode) {
	// 		parent.(*Update).Limit = newNode.(*Limit)
	// 	})
	// case *UpdateExpr:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*UpdateExpr).Name = newNode.(*ColName)
	// 	})
	// 	c.apply(node, n.Expr, func(newNode, parent SQLNode) {
	// 		parent.(*UpdateExpr).Expr = newNode.(Expr)
	// 	})
	// case UpdateExprs:
	// 	for x, el := range n {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(UpdateExprs)[idx] = newNode.(*UpdateExpr)
	// 			}
	// 		}(x))
	// 	}
	// case *Use:
	// 	c.apply(node, n.DBName, func(newNode, parent SQLNode) {
	// 		parent.(*Use).DBName = newNode.(TableIdent)
	// 	})
	// case *VStream:
	// 	c.apply(node, n.Comments, func(newNode, parent SQLNode) {
	// 		parent.(*VStream).Comments = newNode.(Comments)
	// 	})
	// 	c.apply(node, n.SelectExpr, func(newNode, parent SQLNode) {
	// 		parent.(*VStream).SelectExpr = newNode.(SelectExpr)
	// 	})
	// 	c.apply(node, n.Table, func(newNode, parent SQLNode) {
	// 		parent.(*VStream).Table = newNode.(TableName)
	// 	})
	// 	c.apply(node, n.Where, func(newNode, parent SQLNode) {
	// 		parent.(*VStream).Where = newNode.(*Where)
	// 	})
	// 	c.apply(node, n.Limit, func(newNode, parent SQLNode) {
	// 		parent.(*VStream).Limit = newNode.(*Limit)
	// 	})
	// case ValTuple:
	// 	for x, el := range n {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(ValTuple)[idx] = newNode.(Expr)
	// 			}
	// 		}(x))
	// 	}
	// case *Validation:
	// case Values:
	// 	for x, el := range n {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(Values)[idx] = newNode.(ValTuple)
	// 			}
	// 		}(x))
	// 	}
	// case *ValuesFuncExpr:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*ValuesFuncExpr).Name = newNode.(*ColName)
	// 	})
	// case VindexParam:
	// 	c.apply(node, n.Key, replacePanic("VindexParam Key"))
	// case *VindexSpec:
	// 	c.apply(node, n.Name, func(newNode, parent SQLNode) {
	// 		parent.(*VindexSpec).Name = newNode.(ColIdent)
	// 	})
	// 	c.apply(node, n.Type, func(newNode, parent SQLNode) {
	// 		parent.(*VindexSpec).Type = newNode.(ColIdent)
	// 	})
	// 	for x, el := range n.Params {
	// 		c.apply(node, el, func(idx int) func(SQLNode, SQLNode) {
	// 			return func(newNode, container SQLNode) {
	// 				container.(*VindexSpec).Params[idx] = newNode.(VindexParam)
	// 			}
	// 		}(x))
	// 	}
	// case *When:
	// 	c.apply(node, n.Cond, func(newNode, parent SQLNode) {
	// 		parent.(*When).Cond = newNode.(Expr)
	// 	})
	// 	c.apply(node, n.Val, func(newNode, parent SQLNode) {
	// 		parent.(*When).Val = newNode.(Expr)
	// 	})
	case *sqlparser.Where:
		c.apply(node, n.Expr, func(newNode, parent sqlparser.SQLNode) {
			parent.(*sqlparser.Where).Expr = newNode.(sqlparser.Expr)
		})
		// case *XorExpr:
		// 	c.apply(node, n.Left, func(newNode, parent SQLNode) {
		// 		parent.(*XorExpr).Left = newNode.(Expr)
		// 	})
		// 	c.apply(node, n.Right, func(newNode, parent SQLNode) {
		// 		parent.(*XorExpr).Right = newNode.(Expr)
		// 	})
	}

	c.postFunc()

	c.replacer = savedReplacer
	c.node = savedNode
	c.parent = savedParent
}
