package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
)

/*
 * Mapping Int -> Category
 * https://www.abuseipdb.com/categories
 */
var mapping = map[float64]string{
	1:  "DNS Compromise",
	2:  "DNS Poisoning",
	3:  "Fraud Orders",
	4:  "DDoS Attack",
	5:  "FTP Brute-Force",
	6:  "Ping of Death",
	7:  "Phishing",
	8:  "Fraud VoIP",
	9:  "Open Proxy",
	10: "Web Spam",
	11: "Email Spam",
	12: "Blog Spam",
	13: "VPN IP",
	14: "Port Scan",
	15: "Hacking",
	16: "SQL Injection",
	17: "Spoofing",
	18: "Brute-Force",
	19: "Bad Web Bot",
	20: "Exploited Host",
	21: "Web App Attack",
	22: "SSH",
	23: "IoT Targeted",
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
	} else if source.Access["url"][0:4] != "http" {
		return fmt.Errorf("'access.url' must start with 'http[s]://'")
	}

	// Store settings
	p.source = source
	p.limit = limit
	p.url = source.Access["url"]
	p.key = source.Access["key"]

	var err error
	p.maxAgeInDays, err = strconv.Atoi(source.Access["maxAgeInDays"])
	if err != nil {
		return fmt.Errorf("Can't parse 'maxAgeInDays' as integer")
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

	//fmt.Printf("AbuseIPDB %s: %#v\n\n", source.Name, p)
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

	if len(searchFields) == 0 {
		return nil, nil, nil, nil
	}

	var body *bytes.Buffer
	var debug map[string]interface{}

	/*
	 * Send indicators to get results back
	 */
	body, debug, err = p.request(searchFields)
	if err != nil {
		return nil, nil, debug, err
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
	var data map[string]interface{}
	err = json.NewDecoder(body).Decode(&data)
	if err != nil {
		return nil, nil, debug, err
	}

	entries := data["data"].(map[string]interface{})

	mx := sync.Mutex{}
	umx := sync.Mutex{}
	unique := make(map[string]bool)
	counter := 0

	reports := entries["reports"].([]interface{})

	// Process results
	for _, entry := range reports {
		for _, categoryID := range entry.(map[string]interface{})["categories"].([]interface{}) {

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
				stats.Update(entries, field)
			}

			// Go through all the predefined relations and collect unique entries
			for _, relation := range p.source.Relations {
				if entries[relation.From.ID] != nil && entries[relation.To.ID] != nil {
					umx.Lock()

					// Use "Sprintf(...%v..." instead of "entries[relation.From.ID].(string)"
					// as the value can be not a string only
					if _, exists := unique[fmt.Sprintf("%v-%v-%v-%v", relation.From.ID, entries[relation.From.ID], relation.To.ID, entries[relation.To.ID])]; exists {
						if pdk.ResultsContain(results, entries, relation) {
							umx.Unlock()
							continue
						}
					}

					counter++

					unique[fmt.Sprintf("%v-%v-%v-%v", relation.From.ID, entries[relation.From.ID], relation.To.ID, entries[relation.To.ID])] = true
					umx.Unlock()

					/*
					 * Check if expected relation exists in received data.
					 * This allows returned JSON objects to have dynamic schema
					 */
					if _, ok := entries[relation.From.ID]; !ok {
						continue
					}

					if _, ok := entries[relation.To.ID]; !ok {
						continue
					}

					/*
					 * FROM node with attributes
					 */
					from := map[string]interface{}{
						"id":     entries[relation.From.ID],
						"group":  relation.From.Group,
						"search": relation.From.Search,
					}

					// Check FROM type & searching fields
					if len(relation.From.VarTypes) > 0 {
						for _, t := range relation.From.VarTypes {
							if t.RegexCompiled.MatchString(fmt.Sprintf("%v", entries[relation.From.ID])) {
								from["group"] = t.Group
								from["search"] = t.Search

								break
							}
						}
					}

					if len(relation.From.Attributes) > 0 {
						from["attributes"] = make(map[string]interface{})
						pdk.CopyPresentValues(entries, from["attributes"].(map[string]interface{}), relation.From.Attributes)
					}

					/*
					 * TO node
					 */
					to := map[string]interface{}{
						"id":     entries[relation.To.ID],
						"group":  relation.To.Group,
						"search": relation.To.Search,
					}

					// Check FROM type & searching fields
					if len(relation.To.VarTypes) > 0 {
						for _, t := range relation.To.VarTypes {
							if t.RegexCompiled.MatchString(fmt.Sprintf("%v", entries[relation.To.ID])) {
								to["group"] = t.Group
								to["search"] = t.Search

								break
							}
						}
					}

					if len(relation.To.Attributes) > 0 {
						to["attributes"] = make(map[string]interface{})
						pdk.CopyPresentValues(entries, to["attributes"].(map[string]interface{}), relation.To.Attributes)
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
							pdk.CopyPresentValues(entries, result["edge"].(map[string]interface{})["attributes"].(map[string]interface{}), relation.Edge.Attributes)
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

			// Insert custom hardcoded relations.
			// TODO: rewrite this part when plugins development is simplified

			// Comment -> IP
			comment := strings.Trim(entry.(map[string]interface{})["comment"].(string), "\"'")

			from := map[string]interface{}{
				"id":     comment,
				"group":  "comment",
				"search": "comment",
			}

			to := map[string]interface{}{
				"id":     entries["ipAddress"],
				"group":  "ip",
				"search": "ip",
			}

			edge := make(map[string]interface{})
			edge["label"] = "abusive activity"
			edge["attributes"] = map[string]interface{}{
				"reportedAt":          entry.(map[string]interface{})["reportedAt"],
				"reporterCountryCode": entry.(map[string]interface{})["reporterCountryCode"],
				"reporterCountryName": entry.(map[string]interface{})["reporterCountryName"],
				"reporterId":          entry.(map[string]interface{})["reporterId"],
			}

			result := map[string]interface{}{
				"from":   from,
				"to":     to,
				"edge":   edge,
				"source": p.source.Name,
			}

			mx.Lock()
			results = append(results, result)
			mx.Unlock()

			// Category -> Abusive activity
			category, ok := mapping[categoryID.(float64)]
			if !ok {
				category = fmt.Sprintf("%.f", categoryID.(float64))
			}

			umx.Lock()

			if _, exists := unique[fmt.Sprintf("%v-%v-%v-%v", "category", category, "comment", comment)]; exists {
				umx.Unlock()
				continue
			}

			counter++

			unique[fmt.Sprintf("%v-%v-%v-%v", "category", category, "comment", comment)] = true
			umx.Unlock()

			from = map[string]interface{}{
				"id":     category,
				"group":  "taxonomy",
				"search": "taxonomy",
			}

			to = map[string]interface{}{
				"id":     comment,
				"group":  "comment",
				"search": "comment",
			}

			result = map[string]interface{}{
				"from":   from,
				"to":     to,
				"source": p.source.Name,
			}

			mx.Lock()
			results = append(results, result)
			mx.Unlock()
		}
	}

	return results, nil, debug, nil
}

// request connects to the HTTP access point and returns the response
func (p *plugin) request(searchFields [][2]string) (*bytes.Buffer, map[string]interface{}, error) {

	// Create a request body
	data := url.Values{}

	for _, field := range searchFields {
		data.Add(field[0], field[1])
	}

	// Always request for a verbose level
	data.Add("verbose", "")
	data.Add("maxAgeInDays", strconv.Itoa(p.maxAgeInDays))

	var req *http.Request
	var err error

	// Debug info
	debug := make(map[string]interface{})

	// Create a request object
	req, err = http.NewRequest("GET", p.url, nil)
	if err != nil {
		return nil, debug, fmt.Errorf("Can't create a GET request: %s", err.Error())
	}

	req.Header.Add("Key", p.key)
	req.Header.Add("Accept", "application/json")
	req.URL.RawQuery = data.Encode()

	debug["query"] = p.url + "?" + req.URL.RawQuery

	// Declare an HTTP client to execute the request
	client := http.Client{Timeout: p.source.Timeout}

	// Send an HTTP request using a 'req' object
	resp, err := client.Do(req)
	if err != nil {
		return nil, debug, fmt.Errorf("Can't do an HTTP request: %s", err.Error())
	}

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, debug, fmt.Errorf("Can't read an HTTP response: %s", err.Error())
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
