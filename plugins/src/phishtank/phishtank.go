package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
)

type Response struct {
	Results []*Result `xml:"results>url0"`
}

type Result struct {
	URL             string `xml:"url"               json:"url"`
	InDatabase      string `xml:"in_database"       json:"in_database"`
	PhishID         string `xml:"phish_id"          json:"phish_id"`
	PhishDetailPage string `xml:"phish_detail_page" json:"phish_detail_page"`
	Verified        string `xml:"verified"          json:"verified"`
	VerifiedAt      string `xml:"verified_at"       json:"verified_at"`
	Valid           string `xml:"valid"             json:"valid"`
}

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
	} else if source.Access["url"][0:5] != "https" {
		return fmt.Errorf("'access.url' must start with 'https://'")
	}

	// Store settings
	p.source = source
	p.limit = limit
	p.url = source.Access["url"]

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("Phishtank %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {
	return p.source.QueryFields, nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	searchFields, err := p.convert(stmt)
	if err != nil {
		return nil, nil, nil, err
	}

	/*
	 * Send indicators to get results back
	 */
	response, debug, err := p.request(searchFields)
	if err != nil {
		return nil, nil, debug, err
	}

	if response == nil {
		return results, nil, debug, nil
	}

	// Struct to store statistics data
	// when the amount of returned entries is too large
	stats := pdk.NewStats()

	for _, field := range p.source.StatsFields {
		stats.Fields[field] = sortedmap.New(10, desc.Int)
	}

	mx := &sync.Mutex{}
	unique := make(map[string]bool)
	counter := 0

	// Process results
	for _, result := range response.Results {
		var entry map[string]interface{}
		resultByte, _ := json.Marshal(result)
		json.Unmarshal(resultByte, &entry)

		urlParsed, err := url.Parse(result.URL)
		if err != nil {
			return nil, nil, debug, err
		}

		entry["domain"] = urlParsed.Hostname()

		// Stop when results count is too big
		if counter >= p.limit {
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

		pdk.CreateRelations(p.source, entry, unique, &counter, mx, &results)
	}

	return results, nil, debug, nil
}

// request connects to the API access point and returns the response
func (p *plugin) request(searchFields [][2]string) (*Response, map[string]interface{}, error) {

	// API response struct
	var response *Response
	// Debug info
	debug := make(map[string]interface{})

	for _, field := range searchFields {
		// Create a request body
		data := url.Values{}
		data.Add(field[0], field[1])

		var req *http.Request
		var err error

		// Create a request object
		req, err = http.NewRequest("GET", p.url, nil)
		if err != nil {
			return nil, debug, fmt.Errorf("Can't create a GET request: %s", err.Error())
		}

		// Set User-Agent
		if p.source.Access["agent"] != "" {
			req.Header.Set("User-Agent", p.source.Access["agent"])
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded; param=value")
		req.URL.RawQuery = data.Encode()

		debug["query"] = p.url + "?" + req.URL.RawQuery

		// Declare an HTTP client to execute the request
		client := http.Client{Timeout: p.source.Timeout}

		// Send an HTTP request using a 'req' object
		resp, err := client.Do(req)
		if err != nil {
			return nil, debug, fmt.Errorf("Can't do an HTTP request: %s", err.Error())
		}

		buf := &bytes.Buffer{}
		_, err = buf.ReadFrom(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, debug, fmt.Errorf("Can't read an HTTP response: %s", err.Error())
		}
		resp.Body.Close()

		// Check the response
		if resp.StatusCode != http.StatusOK {
			return nil, debug, fmt.Errorf("Bad response StatusCode: %s", resp.Status)
		}

		if bytes.Index(buf.Bytes(), []byte("<in_database>true</in_database>")) > -1 {
			xml.Unmarshal(buf.Bytes(), &response)
			break
		}
	}

	return response, debug, nil
}

func (p *plugin) Stop() error {

	// No error to check, so return nil
	return nil
}
