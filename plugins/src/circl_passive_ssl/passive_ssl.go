package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

func (p *plugin) Conf() *pdk.Source {
	return p.source
}

func (p *plugin) Setup(source *pdk.Source, limit int) error {

	// Validate necessary parameters
	if source.Access["url"] == "" {
		return fmt.Errorf("'access.url' is not defined")
	} else if source.Access["url"][0:4] != "http" {
		return fmt.Errorf("'access.url' must start with 'http[s]://'")
	}

	if source.Access["username"] == "" || source.Access["password"] == "" {
		return fmt.Errorf("Username or password are not defined")
	}

	// Store settings
	p.source = source
	p.limit = limit
	p.url = source.Access["url"]
	p.username = source.Access["username"]
	p.password = source.Access["password"]

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}

		for _, types := range relation.To.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("CIRCL Passive SSL %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {
	return p.source.QueryFields, nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	searchField, err := p.convert(stmt)
	if err != nil {
		return nil, nil, nil, err
	}

	var body *bytes.Buffer
	var debug map[string]interface{}

	/*
	 * Send indicators to get results back
	 */
	body, debug, err = p.request(searchField)
	if err != nil {
		return nil, nil, debug, err
	}

	//fmt.Printf("CIRCL Passive SSL response:\n%v\n", body)

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
	var data map[string]interface{}
	err = json.Unmarshal(body.Bytes(), &data)
	if err != nil {
		return nil, nil, debug, err
	}

	if searchField[0] == "ip" || searchField[0] == "network" {
		for ip, certificates := range data {
			for _, certificate := range certificates.(map[string]interface{})["certificates"].([]interface{}) {
				subjects := data[ip].(map[string]interface{})["subjects"].(map[string]interface{})
				subject, ok := subjects[certificate.(string)].(map[string]interface{})

				if ok {
					values := subject["values"].([]interface{})

					for _, value := range values {
						entry := map[string]interface{}{
							"ip":      ip,
							"sha1":    certificate,
							"subject": value,
						}

						entries = append(entries, entry)
					}
				} else {
					entry := map[string]interface{}{
						"ip":   ip,
						"sha1": certificate,
					}

					entries = append(entries, entry)
				}
			}
		}

	} else if searchField[0] == "sha1" {
		seen := data["seen"].([]interface{})
		for _, ip := range seen {
			entry := map[string]interface{}{
				"ip":   ip,
				"sha1": searchField[1],
			}

			entries = append(entries, entry)
		}
	}

	mx := &sync.Mutex{}
	unique := make(map[string]bool)
	counter := 0

	// Process results
	for _, entry := range entries {

		// Stop when results count is too big
		if counter >= p.limit {
			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return results, top, debug, nil
		}

		// Update stats
		for _, field := range p.source.StatsFields {
			stats.Update(entry, field)
		}

		pdk.CreateRelations(p.source, entry, unique, &counter, mx, &results)
	}

	return results, nil, debug, nil
}

// request connects to the HTTP access point and returns the response
func (p *plugin) request(searchField [2]string) (*bytes.Buffer, map[string]interface{}, error) {

	// Some fields should be processed in a more complex way,
	// so do it here instead of replacing in YAML
	switch searchField[0] {
	case "ip":
		searchField[0] = "query"
		searchField[1] += "/32"
	case "network":
		searchField[0] = "query"
	case "sha1":
		searchField[0] = "cquery"
	}

	// Debug info
	debug := make(map[string]interface{})
	debug["query"] = p.url + "/" + searchField[0] + "/" + searchField[1]

	req, err := http.NewRequest(http.MethodGet, p.url+"/"+searchField[0]+"/"+searchField[1], nil)
	if err != nil {
		return nil, debug, fmt.Errorf("Can't create a GET request: %s", err.Error())
	}

	// Set basic auth credentials if given
	if p.username != "" && p.password != "" {
		req.SetBasicAuth(p.username, p.password)
	}
	req.Header.Set("User-Agent", "graphoscope") // Required parameter

	// Declare an HTTP client to execute the request
	client := http.Client{Timeout: p.source.Timeout}

	// Send an HTTP request using a 'req' object
	resp, err := client.Do(req)
	if err != nil {
		return nil, debug, fmt.Errorf("Can't do a REST API request: %s", err.Error())
	}

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, debug, fmt.Errorf("Can't read a REST API response: %s", err.Error())
	}
	resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return nil, debug, fmt.Errorf("Bad response StatusCode: %s", resp.Status)
	}

	return body, debug, nil
}

func (p *plugin) Stop() error {

	// No error to check, so return nil
	return nil
}
