/*
 * Template to develop new plugins.
 * Check GUI documentation section "Administration" for a step-by-step workflow
 */

package main

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
)

/*
 * Check "pdk/plugin.go" for the built-in plugin functions description
 */

func (p *plugin) Source() *pdk.Source {
	return p.source
}

func (p *plugin) Setup(source *pdk.Source, limit int) error {

	/*
	 * STEP 1.
	 *
	 * Validate required parameters from the YAML config file
	 */

	// if source.Access["url"] == "" {
	// 	return fmt.Errorf("'access.url' is not defined")
	// } else if source.Access["db"] == "" {
	// 	return fmt.Errorf("'access.db' is not defined")
	// }

	/*
	 * STEP 2.
	 *
	 * Create a connection to the data source if needed,
	 * check whether it is established
	 */

	// Be able to cancel execution when
	// DB wants to return > limit amount of entries or time expires
	// ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)
	// defer cancel()

	// client, err := service.Connect(ctx, source.Access["url"], source.Access["db"])
	// if err != nil {
	// 	return err
	// }

	/*
	 * STEP 3.
	 *
	 * Store plugin settings
	 */

	p.source = source
	//p.client = client
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

	return nil
}

func (p *plugin) Fields() ([]string, error) {
	return p.source.QueryFields, nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	/*
	 * STEP 4.
	 *
	 * Convert SQL statement
	 * so the data source can understand what client is searching for.
	 *
	 * Add created query to the debug info, so admin or developer can see
	 * what happens in a background
	 */

	// filter, err := p.convert(stmt)
	// if err != nil {
	// 	return nil, nil, nil, err
	// }

	// Add debug info
	debug := make(map[string]interface{})
	//debug["query"] = searchJSON

	// Context to be able to cancel goroutines
	// when some DB wants to return > limit amount of entries
	// or time expires
	// ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	// defer cancel()

	/*
	 * STEP 6.
	 *
	 * Run the query and get the results.
	 * Here we just create an empty slice for a demo
	 */

	// entries, err := p.client.Find(ctx, filter)
	// if err != nil {
	// 	return nil, nil, debug, err
	// }
	entries := []map[string]interface{}{}

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
	 * STEP 7.
	 *
	 * Process data returned by the data source.
	 * Most of this loop content you shouldn't modify at all.
	 * Some Go packages provide a cursor, so you can use it to go
	 * through the results:
	 *
	 * for cursor.Next() {
	 *     ...
	 * }
	 *
	 * Also uncomment "cancel()"
	 */

	for _, entry := range entries {

		// Stop when results count is over the limit
		if counter >= p.limit {
			// Uncomment in real plugin
			//cancel()

			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return nil, top, debug, nil
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
						pdk.CopyPresentValues(entry, result["edge"].(map[string]interface{}), relation.Edge.Attributes)
					}
				}

				/*
				 * Put it together
				 */
				result["from"] = from
				result["to"] = to
				result["source"] = p.source.Name

				//fmt.Println("Edge:", from, to)

				/*
				 * Add current entry to the list to return
				 */
				mx.Lock()
				results = append(results, result)
				mx.Unlock()
			}
		}
	}

	return results, nil, debug, nil
}

func (p *plugin) Stop() error {
	/*
	 * STEP 8.
	 *
	 * Stop the plugin when main service stops,
	 * drop all connections correctly
	 */

	// ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	// defer cancel()

	// return p.client.Disconnect(ctx)
	return nil
}
