package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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

	// Validate necessary parameters
	if source.Access["url"] == "" {
		return fmt.Errorf("'access.url' is not defined")
	} else if source.Access["url"][0:4] != "http" {
		return fmt.Errorf("'access.url' must start with 'http[s]://'")
	}

	// Store settings
	p.source = source
	p.limit = limit
	p.url = source.Access["url"]

	if strings.ToUpper(source.Access["method"]) == "POST" {
		p.method = "POST"
	} else {
		p.method = "GET"
	}

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}

		for _, types := range relation.To.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("HTTP %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	searchFields, err := p.convert(stmt)
	if err != nil {
		return nil, nil, err
	}

	var body *bytes.Buffer

	/*
	 * Send indicators to get results back
	 */
	body, err = p.request(searchFields)
	if err != nil {
		return nil, nil, err
	}

	// Struct to store statistics data
	// when the amount of returned entries is too large
	stats := pdk.NewStats()

	for _, field := range p.source.StatsFields {
		stats.Fields[field] = sortedmap.New(10, desc.Int)
	}

	/*
	 * Receive hits and deserialize them
	 */
	var entries []map[string]interface{}
	err = json.NewDecoder(body).Decode(&entries)
	if err != nil {
		return nil, nil, err
	}

	mx := sync.Mutex{}
	umx := sync.Mutex{}
	unique := make(map[string]bool)
	counter := 0

	// Process results
	for _, entry := range entries {

		// Stop when results count is too big
		if counter >= p.limit {
			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, err
			}

			return nil, top, nil
		}

		// Update stats
		for _, field := range p.source.StatsFields {
			stats.Update(entry, field)
		}

		// Go through all the predefined relations and collect unique entries
		for _, relation := range p.source.Relations {
			if entry[relation.From.ID] != nil && entry[relation.To.ID] != nil {
				umx.Lock()
				if _, exists := unique[entry[relation.From.ID].(string)+entry[relation.To.ID].(string)]; exists {
					if pdk.ResultsContain(results, entry, relation) {
						umx.Unlock()
						continue
					}
				}

				counter++

				unique[entry[relation.From.ID].(string)+entry[relation.To.ID].(string)] = true
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
						if t.RegexCompiled.MatchString(entry[relation.From.ID].(string)) {
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
						if t.RegexCompiled.MatchString(entry[relation.To.ID].(string)) {
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

				//fmt.Println("Edge:", from, to, p.source.Name)

				/*
				 * Add current entry to the list to return
				 */
				mx.Lock()
				results = append(results, result)
				mx.Unlock()
			}
		}
	}

	return results, nil, nil
}

// request connects to the HTTP access point and returns the response
func (p *plugin) request(searchFields [][2]string) (*bytes.Buffer, error) {

	// Create a request body
	data := url.Values{}

	for _, field := range searchFields {
		data.Add(field[0], field[1])
	}

	var req *http.Request
	var err error

	// Create a request object
	if p.method == "POST" {
		payload := new(bytes.Buffer)
		err = json.NewEncoder(payload).Encode(data)
		if err != nil {
			return nil, fmt.Errorf("Can't encode POST response: %s", err.Error())
		}

		req, err = http.NewRequest("POST", p.url, payload)
		if err != nil {
			return nil, fmt.Errorf("Can't create a POST request: %s", err.Error())
		}

		req.Header.Add("Content-Type", "application/json; charset=UTF-8")

	} else {
		req, err = http.NewRequest("GET", p.url, nil)
		if err != nil {
			return nil, fmt.Errorf("Can't create a POST request: %s", err.Error())
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded; param=value")
		req.URL.RawQuery = data.Encode()
	}

	// Set basic auth credentials if given
	if p.source.Access["user"] != "" && p.source.Access["password"] != "" {
		req.SetBasicAuth(p.source.Access["user"], p.source.Access["password"])
	}

	// Declare an HTTP client to execute the request
	client := http.Client{Timeout: p.source.Timeout}

	// Send an HTTP using `req` object
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Can't do an HTTP request: %s", err.Error())
	}

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, fmt.Errorf("Can't read an HTTP response: %s", err.Error())
	}
	resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bad response StatusCode: %s", resp.Status)
	}

	return body, nil
}

func (p *plugin) Stop() error {

	// No error to check, so return nil
	return nil
}
