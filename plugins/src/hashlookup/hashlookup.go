/*
 * Template to develop new plugins.
 * Check GUI documentation section "Administration" for a step-by-step workflow
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
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

	fmt.Printf("HTTP %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

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
	jsonParsed, err := gabs.ParseJSONBuffer(body)
	if err != nil {
		return nil, nil, err
	}

	entryObj := gabs.New()
	entryObj.Array("entries")

	children := new(gabs.Container)
	parents := new(gabs.Container)

	if jsonParsed.Exists("children") {
		if err != nil {
			return nil, nil, err
		}
		children = jsonParsed.Path("children")
		for _, child := range children.Children() {
			// Add a reference to the parent
			child.Set(jsonParsed.S("SHA-1").Data(), "parent")
			// Add the children to the list of entries
			entryObj.ArrayAppend(child, "entries")
		}
		// Remove the children for the main received object
		jsonParsed.Delete("children")
	}
	if jsonParsed.Exists("parents") {
		parents = jsonParsed.Path("parents")
		if err != nil {
			return nil, nil, err
		}
		for _, child := range parents.Children() {
			// Add a reference to the parent
			child.Set(jsonParsed.S("SHA-1").Data(), "children")
			// Add the children to the list of entries
			entryObj.ArrayAppend(child, "entries")
		}
		// Remove the parents for the main received object
		jsonParsed.Delete("parents")
	}

	entryObj.ArrayAppend(jsonParsed, "entries")

	err = json.NewDecoder(strings.NewReader(entryObj.S("entries").String())).Decode(&entries)
	if err != nil {
		return nil, nil, err
	}

	mx := sync.Mutex{}
	umx := sync.Mutex{}
	unique := make(map[string]bool)
	counter := 0

	for _, entry := range entries {

		// Stop when results count is over the limit
		if counter >= p.limit {
			// Uncomment in real plugin
			//cancel()

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

				//fmt.Println("Edge:", from, to)

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

// TODO implement the other endpoints
// TODO implement bulk ?
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
	tmpUrl := ""
	tmpMethod := ""
	if data.Has("sha1") {
		if len(data["sha1"]) > 1 {
			tmpUrl = p.url + "/bulk/sha1/"
			tmpMethod = "POST"
		} else {
			tmpUrl = p.url + "/lookup/sha1/" + data.Get("sha1")
			tmpMethod = "GET"
		}
	} else {
		fmt.Println("does not have sha1")
	}

	req, err = http.NewRequest(tmpMethod, tmpUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("Can't create a GET request: %s", err.Error())
	}

	req.Header.Add("Content-Type", "application/json")
	req.URL.RawQuery = data.Encode()

	// Set basic auth credentials if given
	// TODO evaluate, irrelevant for the time being
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
	/*
	 * STEP 8.
	 *
	 * Stop the plugin when main service stops,
	 * drop all connections correctly
	 */

	// ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	// defer cancel()

	// return p.client.Disconnect(ctx)
	return nil
}
