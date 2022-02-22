package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
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

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}

		for _, types := range relation.To.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("Pastelyzer %s: %#v\n\n", source.Name, p)
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
	 * Get all artefacts of the requested paste ID
	 */
	if searchFields[0][0] == "source" {
		body, err = p.getArtefacts(searchFields[0][1])
		if err != nil {
			return nil, nil, err
		}

	} else {
		/*
		 * Send indicators to get related paste IDs
		 */
		body, err = p.getPastes(searchFields)
		if err != nil {
			return nil, nil, err
		}
	}

	/*
	 * Receive hits and deserialize them
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

	var entries []map[string]interface{}
	err = json.NewDecoder(body).Decode(&entries)
	if err != nil {
		return nil, nil, err
	}

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

				// Manually add link to the paste's content
				if relation.From.ID == "source" {
					// Show ID only, not full URL
					from["id"] = strings.Split(from["id"].(string), "/")[4]

					// Append additional param,
					// which is no available by default
					if _, ok := from["attributes"]; !ok {
						from["attributes"] = make(map[string]interface{})
					}

					from["attributes"].(map[string]interface{})["content"] = entry["source"].(string) + "body"
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

				// Manually add link to the paste's content
				if relation.To.ID == "source" {
					// Show ID only, not full URL
					to["id"] = strings.Split(to["id"].(string), "/")[4]

					// Append additional param,
					// which is no available by default
					if _, ok := to["attributes"]; !ok {
						to["attributes"] = make(map[string]interface{})
					}

					to["attributes"].(map[string]interface{})["content"] = entry["source"].(string) + "body"
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

				//fmt.Println("Edge:", from, to, p.name)

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

/*
 * Return all artefacts of the given paste ID
 */
func (p *plugin) getArtefacts(id string) (*bytes.Buffer, error) {

	// Create the POST request to the URL with all fields mounted
	req, err := http.NewRequest("GET", p.url+"/content/"+id+"/artefacts/typed", nil)
	if err != nil {
		return nil, fmt.Errorf("Error on request creation: %s", err.Error())
	}

	// Declare an HTTP client to execute the request
	client := http.Client{Timeout: p.source.Timeout}

	// Finally send a POST HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error on request execution: %s", err.Error())
	}

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()

	// Check the response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bad response StatusCode: %s", resp.Status)
	}

	return body, nil
}

/*
 * Return pastes ID where given indicators were found
 */
func (p *plugin) getPastes(searchFields [][2]string) (*bytes.Buffer, error) {

	// Create buffer
	buf := new(bytes.Buffer)
	// Create multipart Writer for that buffer
	w := multipart.NewWriter(buf)

	for _, field := range searchFields {
		if err := w.WriteField(field[0], field[1]); err != nil {
			return nil, err
		}
	}

	// IMPORTANT: Close multipart writer before using it
	w.Close()

	// Create the POST request to the URL with all fields mounted
	req, err := http.NewRequest("POST", p.url+"/artefacts/typed", buf)
	if err != nil {
		return nil, fmt.Errorf("Error on request creation: %s", err.Error())
	}
	// Set the header of the request to send
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Declare our HTTP client to execute the request
	client := http.Client{Timeout: p.source.Timeout}

	// Finally send our POST HTTP request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error on request execution: %s", err.Error())
	}

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, err
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
