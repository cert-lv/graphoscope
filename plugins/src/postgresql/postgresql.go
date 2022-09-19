package main

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4/pgxpool"
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

	// Validate necessary parameters
	if source.Access["user"] == "" {
		return fmt.Errorf("'access.user' is not defined")
	} else if source.Access["password"] == "" {
		return fmt.Errorf("'access.password' is not defined")
	} else if source.Access["addr"] == "" {
		return fmt.Errorf("'access.addr' is not defined")
	} else if source.Access["db"] == "" {
		return fmt.Errorf("'access.db' is not defined")
	} else if source.Access["table"] == "" {
		return fmt.Errorf("'access.table' is not defined")
	}

	// URL-encode password in case it contains special characters,
	// which will break the configuration string.
	// Also Golang encodes space to + sign. We need %20 instead
	password := strings.ReplaceAll(url.QueryEscape(source.Access["password"]), "+", "%20")

	config, err := pgxpool.ParseConfig("postgres://" + source.Access["user"] + ":" + password + "@" + source.Access["addr"] + "/" + source.Access["db"])
	if err != nil {
		return err
	}

	//config.TLSConfig.ServerName = "localhost"
	config.ConnConfig.PreferSimpleProtocol = true
	config.MaxConns = 32

	// Be able to cancel too long execution
	ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)
	defer cancel()

	// Connect to the database
	conn, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return err
	}

	// Check the connection
	err = conn.Ping(ctx)
	if err != nil {
		return err
	}

	// Store settings
	p.source = source
	p.connection = conn
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

	//fmt.Printf("PostgreSQL %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {

	// Slice to store the search result
	entries := []map[string]interface{}{}

	// Context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	// Request 1 row to get all the possible columns
	err := pgxscan.Select(ctx, p.connection, &entries, "SELECT * FROM "+p.source.Access["table"]+" LIMIT 1")
	if err != nil {
		return nil, err
	}

	// Convert map to the slice
	fields := make([]string, 0, len(entries[0]))
	for value, _ := range entries[0] {
		fields = append(fields, value)
	}

	return fields, nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	filter, err := p.convert(stmt)
	if err != nil {
		return nil, nil, nil, err
	}

	query := "SELECT " + sqlparser.String(stmt.SelectExprs) + " FROM " + p.source.Access["table"] + " WHERE " + filter

	// Debug info
	debug := make(map[string]interface{})
	debug["query"] = query

	/*
	 * Run the query
	 */

	// Context to be able to cancel goroutines
	// when some DB wants to return > limit amount of entries
	// or time expires
	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	// Slice to store all the search results
	entries := []map[string]interface{}{}

	err = pgxscan.Select(ctx, p.connection, &entries, query)
	if err != nil {
		return nil, nil, debug, err
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

	for _, entry := range entries {

		// Stop when results count is too big
		if counter >= p.limit {
			cancel()

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
	p.connection.Close()
	return nil
}
