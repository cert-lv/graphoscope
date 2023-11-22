package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/redis/go-redis/v9"
)

/*
 * Check "pdk/plugin.go" for the built-in plugin functions description
 */

func (p *plugin) Conf() *pdk.Source {
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
	}

	intDB, err := strconv.Atoi(source.Access["db"])
	if err != nil {
		return fmt.Errorf("'access.db' is not defined as an integer")
	}

	options := &redis.Options{
		Addr:     source.Access["addr"],
		Username: source.Access["user"],
		Password: source.Access["password"],
		DB:       intDB,
		// TLSConfig: &tls.Config{
		//     MinVersion:   tls.VersionTLS12,
		//     Certificates: []tls.Certificate{cert},
		//     RootCAs:      caCertPool,
		// },
	}

	client := redis.NewClient(options)

	// Be able to cancel too long execution
	ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)
	defer cancel()

	// Check the connection
	pong := client.Ping(ctx)
	if pong.String() != "ping: PONG" {
		return fmt.Errorf("%v", pong)
	}

	// Store settings
	p.source = source
	p.client = client
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

	//fmt.Printf("Redis %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {
	return p.source.QueryFields, nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	filter, err := p.convert(stmt)
	if err != nil {
		return nil, nil, nil, err
	}

	// Debug info
	debug := make(map[string]interface{})
	debug["query"] = p.source.Access["field"] + ":" + filter

	/*
	 * Run the query
	 */

	// Context to be able to cancel goroutines when time expires
	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	record := p.client.HGetAll(ctx, filter).Val()
	entry := map[string]interface{}{
		p.source.Access["field"]: filter,
	}

	for k, v := range record {
		entry[k] = v
	}

	// Go through all the predefined relations and create them
	for _, relation := range p.source.Relations {
		fromID, fromExists := entry[relation.From.ID]
		toID, toExists := entry[relation.To.ID]

		if fromExists && toExists {

			/*
			 * FROM node with attributes
			 */
			from := map[string]interface{}{
				"id":     fromID,
				"group":  relation.From.Group,
				"search": relation.From.Search,
			}

			// Check FROM type & searching fields
			if len(relation.From.VarTypes) > 0 {
				for _, t := range relation.From.VarTypes {
					if t.RegexCompiled.MatchString(fmt.Sprintf("%v", fromID)) {
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
				"id":     toID,
				"group":  relation.To.Group,
				"search": relation.To.Search,
			}

			// Check FROM type & searching fields
			if len(relation.To.VarTypes) > 0 {
				for _, t := range relation.To.VarTypes {
					if t.RegexCompiled.MatchString(fmt.Sprintf("%v", toID)) {
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

			//fmt.Println("Edge:", from, to)

			/*
			 * Add current entry to the list to return
			 */
			results = append(results, result)
		}
	}

	return results, nil, debug, nil
}

func (p *plugin) Stop() error {
	if p.client != nil {
		p.client.Close()
	}

	return nil
}
