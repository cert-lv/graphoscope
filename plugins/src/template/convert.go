/*
 * SQL to X query convertor
 */

package main

import (
	"github.com/blastrain/vitess-sqlparser/sqlparser"
)

/*
 * Convert SQL statement to the object expected by the data source
 */
func (p *plugin) convert(sel *sqlparser.Select) (string, error) {

	/*
	 * STEP 8.
	 *
	 * Do the SQL conversion.
	 * Check, for example, a MongoDB plugin to see how SQL
	 * can be converted to the hierarchical object.
	 *
	 * Here we just return a simple static 'field=value' pair.
	 *
	 * File not needed for the processor plugin!
	 */

	filter := "field=value"
	return filter, nil
}
