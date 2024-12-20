package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/elastic/go-elasticsearch/v7"
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
	cert, err := ioutil.ReadFile(source.Access["ca"])
	if err != nil {
		fmt.Errorf("Unable to read CA from %q: %s", source.Access["ca"], err)
	}

	// Several ways to authorize the user
	cfg := elasticsearch.Config{
		Addresses: []string{source.Access["url"]},
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: source.Timeout,
			DialContext: (&net.Dialer{
				Timeout:   source.Timeout,
				KeepAlive: source.Timeout,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
		CACert: cert,
	}

	if source.Access["key"] != "" {
		cfg.APIKey = source.Access["key"]
	} else if source.Access["username"] != "" && source.Access["password"] != "" {
		cfg.Username = source.Access["username"]
		cfg.Password = source.Access["password"]
	}

	// Elasticsearch server address
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return err
	}

	// Ping the Elasticsearch server to get e.g. the version number
	_, err = client.Info()
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
	mappings, err := p.client.Indices.GetMapping(p.client.Indices.GetMapping.WithIndex(p.index))
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(mappings.Body)
	if err != nil {
		return nil, err
	}

	data := make(map[string]map[string]map[string]map[string]interface{})
	json.Unmarshal(body, &data)

	for _, v := range data {
		for _, props := range v {
			for _, list := range props {
				for field, _ := range list {
					fieldsMap[field] = true
				}
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
	found, err := p.client.Search(
		p.client.Search.WithIndex(p.index),
		p.client.Search.WithBody(strings.NewReader(searchJSON)),
		p.client.Search.WithContext(ctx),
	)
	if err != nil {
		return nil, nil, debug, err
	}

	b, err := io.ReadAll(found.Body)
	if err != nil {
		return nil, nil, debug, err
	}

	response := make(map[string]map[string][]map[string]interface{})
	json.Unmarshal(b, &response)

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
	for _, hit := range response["hits"]["hits"] {

		// Stop when results count is too big
		if counter >= p.limit {
			cancel()

			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return results, top, debug, nil
		}

		entry, ok := hit["_source"].(map[string]interface{})
		if !ok {
			return nil, nil, debug, fmt.Errorf("Can't decode '_source' response field")
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
