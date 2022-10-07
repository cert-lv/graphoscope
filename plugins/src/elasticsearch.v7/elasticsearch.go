package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/olivere/elastic/v7"
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
	if source.Access["url"] == "" {
		return fmt.Errorf("'access.url' is not defined")
	} else if source.Access["url"][0:4] != "http" {
		return fmt.Errorf("'access.url' must start with 'http[s]://'")
	} else if source.Access["indices"] == "" {
		return fmt.Errorf("'access.indices' is not defined")
	}

	// Several ways to authorize the user
	header := http.Header{}

	if source.Access["key"] != "" {
		header["Authorization"] = []string{"ApiKey " + source.Access["key"]}
	} else if source.Access["username"] != "" && source.Access["password"] != "" {
		header["Authorization"] = []string{"Basic " + base64.StdEncoding.EncodeToString([]byte(source.Access["username"]+":"+source.Access["password"]))}
	}

	// Elasticsearch server address
	client, err := elastic.NewClient(
		elastic.SetHeaders(header),
		elastic.SetURL(source.Access["url"]),
		elastic.SetSniff(false))
	if err != nil {
		return err
	}

	// Starting with elastic.v5, you must pass a context to execute each service
	// to be able to cancel execution when
	// some DB wants to return > limit amount of entries or time expires
	ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)
	defer cancel()

	// Ping the Elasticsearch server to get e.g. the version number
	_, _, err = client.Ping(source.Access["url"]).Do(ctx)
	if err != nil {
		return err
	}

	// Store settings
	p.source = source
	p.client = client
	p.limit = limit
	p.index = source.Access["indices"]

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}

		for _, types := range relation.To.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("ES %s: %#v\n\n", p.source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {

	// Map for collecting unique fields only
	fieldsMap := make(map[string]bool)

	// In elasticsearch mapping is a place to get all the fields from
	mapping := p.client.GetMapping()
	service := mapping.Index(p.index)

	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	results, err := service.Do(ctx)
	if err != nil {
		return nil, err
	}

	// Recursively process all the fields
	for _, result := range results {
		properties := result.(map[string]interface{})["mappings"].(map[string]interface{})["properties"]
		if properties == nil {
			// Additional option for campatibility with Elasticsearch 6.x
			event := result.(map[string]interface{})["mappings"].(map[string]interface{})["event"]
			if event != nil {
				properties = result.(map[string]interface{})["mappings"].(map[string]interface{})["event"].(map[string]interface{})["properties"]
			}
		}

		switch prop := properties.(type) {
		case map[string]interface{}:
			for field, data := range prop {
				if data.(map[string]interface{})["properties"] != nil {
					p.getFields(data.(map[string]interface{})["properties"].(map[string]interface{}), field, fieldsMap)
				} else {
					fieldsMap[field] = true
				}
			}
		}
	}

	// Convert map to the slice
	fields := make([]string, 0, len(fieldsMap))
	for value, _ := range fieldsMap {
		fields = append(fields, value)
	}

	return fields, nil
}

func (p *plugin) getFields(data map[string]interface{}, field string, fields map[string]bool) {
	for f, m := range data {
		if m.(map[string]interface{})["properties"] != nil {
			p.getFields(m.(map[string]interface{})["properties"].(map[string]interface{}), field+"."+f, fields)
		} else {
			fields[field+"."+f] = true
		}
	}
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	searchJSON, err := p.convert(stmt, p.source.IncludeFields)
	if err != nil {
		return nil, nil, nil, err
	}

	// Debug info
	debug := make(map[string]interface{})
	debug["query"] = searchJSON

	// Context to be able to cancel goroutines
	// when some DB wants to return > limit amount of entries
	// or time expires
	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	// Search in Elasticsearch using a raw JSON string.
	// Will not return more than 10 000 entries with a default Elasticsearch.
	// "scroll" is not being used here, because LIMIT's offset can't bet set:
	//
	// elastic: Error 400 (Bad Request): Validation Failed: 1: using [from] is not allowed in a scroll context
	//
	// So use a single request to make the plugin consistent with
	// the other plugins
	found, err := p.client.Search().Index(p.index).Source(searchJSON).Do(ctx)
	if err != nil {
		return nil, nil, debug, err
	}

	/*
	 * Concurrent goroutines receive hits and deserialize them.
	 * Number is set by "runtime.NumCPU()" - host's CPU cores count
	 */

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

	// Iterate through the results
	for _, hit := range found.Hits.Hits {

		// Stop when results count is too big
		if counter >= p.limit {
			cancel()

			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return nil, top, debug, nil
		}

		// Deserialize
		entry := make(map[string]interface{})

		if len(p.source.IncludeFields) != 0 {
			for key, value := range hit.Fields {
				entry[key] = value.([]interface{})[0]
			}
		} else {
			err := json.Unmarshal(hit.Source, &entry)
			if err != nil {
				return nil, nil, debug, err
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

		// Terminate early?
		select {
		case <-ctx.Done():
			// Parsing ES search results canceled
			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return nil, top, debug, nil
		default:
		}
	}

	return results, nil, debug, nil
}

func (p *plugin) Stop() error {
	if p.client != nil {
		p.client.Stop()
	}

	// No error to check, so return nil
	return nil
}
