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

// type Response struct {
// 	IP       string      `json:"ip"`
// 	Hostname string      `json:"hostname"`
// 	Anycast  bool        `json:"anycast"`
// 	City     string      `json:"city"`
// 	Region   string      `json:"region"`
// 	Country  string      `json:"country"`
// 	Loc      string      `json:"loc"`
// 	Org      string      `json:"org"`
// 	Postal   string      `json:"postal"`
// 	Timezone string      `json:"timezone"`
// 	Bogon    bool        `json:"bogon"`
// 	Status   string      `json:"status"`
// 	Error    interface{} `json:"error"`
// }

/*
 * Check "pdk/plugin.go" for the built-in plugin functions description
 */

func (p *plugin) Conf() *pdk.Source {
	return p.source
}

func (p *plugin) Setup(source *pdk.Source, limit int) error {

	// Validate necessary parameters
	if source.Access["server"] == "" {
		return fmt.Errorf("'access.server' is not defined")
	} else if source.Access["server"][0:5] != "https" {
		return fmt.Errorf("'access.server' must start with 'https://'")
	}

	if source.Access["server"][len(source.Access["server"])-1:] != "/" {
		source.Access["server"] += "/"
	}

	// Store settings
	p.source = source
	p.limit = limit
	p.server = source.Access["server"]
	p.token = source.Access["token"]

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("Ipinfo %s: %#v\n\n", source.Name, p)
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

	// With a free access level only IP can be queried
	if searchField[0] != "ip" {
		return results, nil, nil, nil
	}

	/*
	 * Send indicators to get results back
	 */
	response, debug, err := p.request(searchField)
	if err != nil {
		return nil, nil, debug, err
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

	// API response struct
	var entry map[string]interface{}
	err = json.Unmarshal(response, &entry)
	if err != nil {
		return nil, nil, debug, err
	}

	if entry["error"] != nil {
		return nil, nil, debug, fmt.Errorf("%v", entry["error"].(map[string]interface{})["message"].(string))
	}

	// Separate ASN number from Org name
	if entry["org"] != nil {
		match, _ := regexp.MatchString("^AS\\d* ", entry["org"].(string))
		if match {
			orgN := strings.SplitN(entry["org"].(string), " ", 2)
			entry["asn"] = orgN[0]
			entry["org"] = orgN[1]
		}
	}

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

	return results, nil, debug, nil
}

// request connects to the API access point and returns the response
func (p *plugin) request(searchField [2]string) ([]byte, map[string]interface{}, error) {

	// Debug info
	debug := make(map[string]interface{})

	// Create a request body
	var req *http.Request
	var err error

	// Create a request object
	req, err = http.NewRequest("GET", p.server+searchField[1]+"/json", nil)
	if err != nil {
		return nil, debug, fmt.Errorf("Can't create a GET request: %s", err.Error())
	}

	// Set access token
	data := url.Values{}
	if p.source.Access["token"] != "" {
		data.Add("token", p.source.Access["token"])
	}

	req.URL.RawQuery = data.Encode()
	debug["query"] = p.server + searchField[1] + "/json"

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

	debug["response"] = string(buf.Bytes())
	return buf.Bytes(), debug, nil
}

func (p *plugin) Stop() error {

	// No error to check, so return nil
	return nil
}
