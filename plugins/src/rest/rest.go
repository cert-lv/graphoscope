package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

	// fmt.Printf("REST %s: %#v\n\n", source.Name, p)
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

	//fmt.Printf("REST API response:\n%v\n", body)

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

	//API response is a list of JSONs not one JSON object. Prior unmarshaling it is needed to fix JSON format
	s := body.String() // Byte array to string

	dataRaw := strings.Split(s, "\n") // Slice from multiline string
	data := dataRaw[:len(dataRaw)-1]  // Delete last empty line

	//Adding JSON delimiters and comma separation
	var buffer bytes.Buffer
	buffer.WriteString("[") //starting JSON char
	for i, rec := range data {
		buffer.WriteString(rec) //trailing JSON char
		if i < len(data)-1 {    //add comma delimiter except last iteration
			buffer.WriteString(",")
		}
	}
	buffer.WriteString("]")     //trailing JSON char
	jsonData := buffer.String() //final valid JSON string ready for unmarshaling

	err = json.Unmarshal([]byte(jsonData), &entries)
	if err != nil {
		return nil, nil, debug, err
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

	// Debug info
	debug := make(map[string]interface{})

	// Declare an HTTP client to execute the request
	client := http.Client{Timeout: p.source.Timeout}

	debug["query"] = p.url + "/" + searchField[0] + "/" + searchField[1]

	req, err := http.NewRequest(http.MethodGet, p.url+"/"+searchField[0]+"/"+searchField[1], nil)
	if err != nil {
		return nil, debug, fmt.Errorf("Can't create a GET request: %s", err.Error())
	}

	// Set basic auth credentials if given
	if p.username != "" && p.password != "" {
		req.SetBasicAuth(p.username, p.password)
	}
	req.Header.Set("User-Agent", "graphoscope") // Required parameter for some APIs

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
