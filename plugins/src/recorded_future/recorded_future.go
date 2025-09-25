package main

import (
	"bytes"
	"encoding/json"
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
	Data  *Data  `json:"data"`
	Error *Error `json:"error"`
}

type Data struct {
	Risk       *Risk       `json:"risk"`
	AiInsights *AiInsights `json:"aiInsights"`
	Categories []*Category `json:"categories"`
	Entity     *Entity     `json:"entity"`
	Timestamps *Timestamps `json:"timestamps"`
}

type Error struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
	Status  int    `json:"status"`
}

type AiInsights struct {
	Comment            string `json:"comment"`
	Text               string `json:"text"`
	NumberOfReferences int    `json:"numberOfReferences"`
}

type Risk struct {
	CriticalityLabel string            `json:"criticalityLabel"`
	RiskString       string            `json:"riskString"`
	Rules            int               `json:"rules"`
	Criticality      int               `json:"criticality"`
	RiskSummary      string            `json:"riskSummary"`
	Score            int               `json:"score"`
	EvidenceDetails  []*EvidenceDetail `json:"evidenceDetails"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Entity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Timestamps struct {
	LastSeen  string `json:"lastSeen"`
	FirstSeen string `json:"firstSeen"`
}

type EvidenceDetail struct {
	MitigationString string `json:"mitigationString"`
	EvidenceString   string `json:"evidenceString"`
	Rule             string `json:"rule"`
	Criticality      int    `json:"criticality"`
	Timestamp        string `json:"timestamp"`
	CriticalityLabel string `json:"criticalityLabel"`
}

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
	p.fields = source.QueryFields
	p.token = source.Access["token"]

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("Recorded Future %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {
	return p.source.QueryFields, nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}
	// Debug info
	debug := make(map[string]interface{})

	// Convert SQL statement
	searchField, err := p.convert(stmt)
	if err != nil {
		return nil, nil, nil, err
	}

	// Continue if requested field is acceptable
	// according to your subscription level
	// and is defined in a YAML file in 'queryFields'
	acceptable := false
	for _, field := range p.fields {
		if searchField[0] == field {
			acceptable = true
			continue
		}
	}
	if !acceptable {
		debug["unacceptable field"] = searchField[0]
		debug["value"] = searchField[1]

		return results, nil, debug, err
	}

	/*
	 * Send indicators to get results back
	 */
	b, debug, err := p.request(searchField)
	if err != nil {
		return nil, nil, debug, err
	}

	response := &Response{}
	err = json.Unmarshal(b, response)
	if err != nil {
		return nil, nil, debug, err
	}

	if response.Error != nil {
		return nil, nil, debug, fmt.Errorf("%v %v", response.Error.Status, response.Error.Message)
	}

	entries := []map[string]interface{}{}
	if response.Data.Risk != nil {
		for _, evidence := range response.Data.Risk.EvidenceDetails {
			entry := map[string]interface{}{
				searchField[0]: searchField[1],

				"criticalityLabel": response.Data.Risk.CriticalityLabel,
				"riskString":       response.Data.Risk.RiskString,
				"rules":            response.Data.Risk.Rules,
				"criticality":      response.Data.Risk.Criticality,
				"riskSummary":      response.Data.Risk.RiskSummary,
				"score":            response.Data.Risk.Score,

				"mitigationString":         evidence.MitigationString,
				"evidenceString":           evidence.EvidenceString,
				"rule":                     evidence.Rule,
				"evidenceCriticality":      evidence.Criticality,
				"timestamp":                evidence.Timestamp,
				"evidenceCriticalityLabel": evidence.CriticalityLabel,

				"lastSeen":  response.Data.Timestamps.LastSeen,
				"firstSeen": response.Data.Timestamps.FirstSeen,
			}

			if response.Data.AiInsights != nil {
				entry["aiInsightsComment"] = response.Data.AiInsights.Comment
				entry["aiInsights"] = response.Data.AiInsights.Text
			}

			entries = append(entries, entry)
		}
	}

	for _, category := range response.Data.Categories {
		entry := map[string]interface{}{
			category.Type: category.Name,

			"malwareID": response.Data.Entity.ID,
			"malware":   response.Data.Entity.Name,

			"lastSeen":  response.Data.Timestamps.LastSeen,
			"firstSeen": response.Data.Timestamps.FirstSeen,
		}

		if response.Data.AiInsights != nil {
			entry["aiInsightsComment"] = response.Data.AiInsights.Comment
			entry["aiInsights"] = response.Data.AiInsights.Text
		}

		entries = append(entries, entry)
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

// request connects to the API access point and returns the response
func (p *plugin) request(searchField [2]string) ([]byte, map[string]interface{}, error) {

	// Debug info
	debug := make(map[string]interface{})

	// Create a request body
	var req *http.Request
	var err error

	// Create a request object
	req, err = http.NewRequest("GET", p.server+searchField[0]+"/"+url.QueryEscape(searchField[1]), nil)
	if err != nil {
		return nil, debug, fmt.Errorf("Can't create a GET request: %s", err.Error())
	}

	// Set access token
	req.Header.Set("X-RFToken", p.source.Access["token"])

	// Request specific info/fields, comma separated
	data := url.Values{}
	data.Add("fields", "entity,risk,aiInsights,categories,timestamps")
	req.URL.RawQuery = data.Encode()

	debug["query"] = req.URL.Scheme + "://" + req.URL.Host
	if req.URL.RawPath != "" {
		debug["query"] = debug["query"].(string) + req.URL.RawPath
	} else {
		debug["query"] = debug["query"].(string) + req.URL.Path
	}
	debug["query"] = debug["query"].(string) + "?" + req.URL.RawQuery

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

	if len(buf.Bytes()) != 0 {
		debug["response"] = string(buf.Bytes())
	}

	return buf.Bytes(), debug, nil
}

func (p *plugin) Stop() error {

	// No error to check, so return nil
	return nil
}
