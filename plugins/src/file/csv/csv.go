package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	_ "github.com/mithrandie/csvq-driver"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
)

/*
 * Check "pdk/plugin.go" for the built-in plugin functions description
 */

func (p *plugin) Conf() *pdk.Source {
	return p.source
}

func (p *plugin) Setup(source *pdk.Source, limit int) error {

	// Validate necessary parameters
	if source.Access["path"] == "" {
		return fmt.Errorf("'access.path' is not defined")
	}

	// Get directory of the data file (will be used as a database)
	// and fielname (will be used as a table)
	p.dir = filepath.Dir(source.Access["path"])
	p.base = filepath.Base(source.Access["path"])

	db, err := sql.Open("csvq", p.dir)
	if err != nil {
		return err
	}

	// Be able to cancel too long execution
	ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)
	defer cancel()

	// Check the connection
	err = db.PingContext(ctx)
	if err != nil {
		return err
	}

	// Store settings
	p.source = source
	p.db = db
	p.limit = limit

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}

		for _, types := range relation.To.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("CSV %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {

	// Request 1 row to get all the possible columns.
	// Additional variable to prevent "gosec" tool's warning:
	// G202: SQL string concatenation
	query := "SELECT * FROM `" + p.base + "` LIMIT 1"
	rows, err := p.db.Query(query)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	return cols, nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	filter, err := p.convert(stmt)
	if err != nil {
		return nil, nil, nil, err
	}

	query := "SELECT " + sqlparser.String(stmt.SelectExprs) + " FROM `" + p.base + "` WHERE " + filter

	// Debug info
	debug := make(map[string]interface{})
	debug["query"] = query

	/*
	 * Run the query
	 */

	// Context to be able to cancel goroutines
	// when DB wants to return > limit amount of entries or time expires
	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, debug, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, debug, err
	}

	var row = make([]interface{}, 0, len(cols))
	for range cols {
		// Use "sql.NullString" instead of "string" to be able
		// to handle empty values in a CSV file
		row = append(row, new(sql.NullString))
	}

	mx := sync.Mutex{}
	umx := sync.Mutex{}
	unique := make(map[string]bool)
	counter := 0

	// Struct to store statistics data
	// when the amount of returned entries is too large
	stats := pdk.NewStats()

	for _, field := range p.source.StatsFields {
		stats.Fields[field] = sortedmap.New(10, desc.Int)
	}

	/*
	 * Iterate through the results
	 */

	for rows.Next() {

		// Stop when results count is too big
		if counter >= p.limit {
			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return nil, top, debug, nil
		}

		if err := rows.Scan(row...); err != nil {
			return nil, nil, debug, err
		}

		// Deserialize
		entry := make(map[string]interface{})

		for i, col := range cols {
			value, _ := row[i].(*sql.NullString).Value()
			if value != nil {
				entry[col] = value.(string)
			}
		}

		// Update stats
		for _, field := range p.source.StatsFields {
			stats.Update(entry, field)
		}

		// Go through all the predefined relations and collect unique entries
		for _, relation := range p.source.Relations {
			if entry[relation.From.ID] != nil && entry[relation.To.ID] != nil {
				umx.Lock()

				// Use "Printf(...%v..." instead of "entry[relation.From.ID].(string)"
				// as the value can be not a string only
				if _, exists := unique[fmt.Sprintf("%v-%v-%v-%v", relation.From.ID, entry[relation.From.ID], relation.To.ID, entry[relation.To.ID])]; exists {
					if pdk.ResultsContain(results, entry, relation) {
						umx.Unlock()
						continue
					}
				}

				counter++

				unique[fmt.Sprintf("%v-%v-%v-%v", relation.From.ID, entry[relation.From.ID], relation.To.ID, entry[relation.To.ID])] = true
				umx.Unlock()

				/*
				 * FROM node with attributes
				 */
				from := map[string]interface{}{
					"id":     entry[relation.From.ID],
					"group":  relation.From.Group,
					"search": relation.From.Search,
				}

				// Check FROM type & searching fields
				if len(relation.From.VarTypes) > 0 {
					for _, t := range relation.From.VarTypes {
						if t.RegexCompiled.MatchString(fmt.Sprintf("%v", entry[relation.From.ID])) {
							from["group"] = t.Group
							from["search"] = t.Search

							break
						}
					}
				}

				if len(relation.From.Attributes) > 0 {
					from["attributes"] = make(map[string]interface{})
					pdk.CopyPresentValues(entry, from["attributes"].(map[string]interface{}), relation.From.Attributes)
				}

				/*
				 * TO node
				 */
				to := map[string]interface{}{
					"id":     entry[relation.To.ID],
					"group":  relation.To.Group,
					"search": relation.To.Search,
				}

				// Check FROM type & searching fields
				if len(relation.To.VarTypes) > 0 {
					for _, t := range relation.To.VarTypes {
						if t.RegexCompiled.MatchString(fmt.Sprintf("%v", entry[relation.To.ID])) {
							to["group"] = t.Group
							to["search"] = t.Search

							break
						}
					}
				}

				if len(relation.To.Attributes) > 0 {
					to["attributes"] = make(map[string]interface{})
					pdk.CopyPresentValues(entry, to["attributes"].(map[string]interface{}), relation.To.Attributes)
				}

				// Resulting graph entry to return
				result := make(map[string]interface{})

				/*
				 * Edge between FROM and TO
				 */
				if relation.Edge != nil && (relation.Edge.Label != "" || len(relation.Edge.Attributes) > 0) {
					result["edge"] = make(map[string]interface{})

					if relation.Edge.Label != "" {
						result["edge"].(map[string]interface{})["label"] = relation.Edge.Label
					}

					if len(relation.Edge.Attributes) > 0 {
						result["edge"].(map[string]interface{})["attributes"] = make(map[string]interface{})
						pdk.CopyPresentValues(entry, result["edge"].(map[string]interface{})["attributes"].(map[string]interface{}), relation.Edge.Attributes)
					}
				}

				/*
				 * Put it together
				 */
				result["from"] = from
				result["to"] = to
				result["source"] = p.source.Name

				// fmt.Println("Edge:", from, to, p.source.Name)

				/*
				 * Add current entry to the list to return
				 */
				mx.Lock()
				results = append(results, result)
				mx.Unlock()
			}
		}
	}

	err = rows.Err()
	if err != nil {
		return nil, nil, debug, err
	}

	return results, nil, debug, nil
}

func (p *plugin) Stop() error {
	if p.db == nil {
		return nil
	}

	return p.db.Close()
}
