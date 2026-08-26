package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
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
	if source.Access["url"] == "" {
		return fmt.Errorf("'access.url' is not defined")
	} else if source.Access["url"][0:4] != "http" {
		return fmt.Errorf("'access.url' must start with 'http[s]://'")
	} else if source.Access["indices"] == "" {
		return fmt.Errorf("'access.indices' is not defined")
	}

	// Read the CA from file
	cert, err := os.ReadFile(source.Access["ca"])
	if err != nil {
		return fmt.Errorf("Unable to read CA from %q: %s", source.Access["ca"], err)
	}

	opts := []elasticsearch.Option{
		elasticsearch.WithAddresses(source.Access["url"]),
	}

	if len(cert) > 0 {
		opts = append(opts, elasticsearch.WithCACert(cert))
	}

	// Several ways to authorize the user
	if source.Access["key"] != "" {
		opts = append(opts, elasticsearch.WithAPIKey(source.Access["key"]))
	} else if source.Access["username"] != "" && source.Access["password"] != "" {
		opts = append(opts, elasticsearch.WithBasicAuth(source.Access["username"], source.Access["password"]))
	}

	client, err := elasticsearch.NewTyped(opts...)
	if err != nil {
		return err
	}

	// Ping the Elasticsearch server to get e.g. the version number
	ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)
	defer cancel()

	_, err = client.Ping().Do(ctx)
	if err != nil {
		return fmt.Errorf("Failed to ping Elasticsearch cluster: %s", err.Error())
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

	// fmt.Printf("ESv9 %s: %#v\n\n", p.source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	mappings, err := p.client.Indices.GetMapping().
		Index(p.index).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	// Map for collecting unique fields only
	fieldsMap := make(map[string]bool)

	// Extract top-level and nested field names
	for _, mapping := range mappings {
		indexFields := p.getFields(mapping.Mappings.Properties, "")
		for _, field := range indexFields {
			if _, exists := fieldsMap[field]; !exists {
				fieldsMap[field] = true
			}
		}
	}

	// Convert map to the slice
	fields := make([]string, 0, len(fieldsMap))
	for value := range fieldsMap {
		fields = append(fields, value)
	}

	return fields, nil
}

// Helper function to recursively collect all property paths
func (p *plugin) getFields(properties map[string]types.Property, prefix string) []string {

	var fields []string

	for name, prop := range properties {
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}
		fields = append(fields, fullName)

		// Get sub-properties for Object or Nested property types
		var subProps map[string]types.Property

		switch p := prop.(type) {
		case *types.ObjectProperty:
			subProps = p.Properties
		case *types.NestedProperty:
			subProps = p.Properties
		}

		// Recurse if nested object properties exist
		if len(subProps) > 0 {
			fields = append(fields, p.getFields(subProps, fullName)...)
		}
	}

	return fields
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
	// "scroll" is not being used here.
	found, err := p.client.Search().
		Index(p.index).
		Raw(strings.NewReader(searchJSON)).
		Do(ctx)
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

			return results, top, debug, nil
		}

		var entry map[string]interface{}

		err := json.Unmarshal(hit.Source_, &entry)
		if err != nil {
			return nil, nil, debug, fmt.Errorf("Failed to unmarshal hit %s: %s", hit.Id_, err.Error())
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
	// No error to check, so return nil
	return nil
}
