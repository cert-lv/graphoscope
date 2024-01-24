/*
 * Template to develop new plugins.
 * Check GUI documentation section "Administration" for a step-by-step workflow
 */

package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}

		for _, types := range relation.To.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// TODO check that
	fmt.Printf("HTTP %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Debug info
	debug := make(map[string]interface{})

	searchFields, err := p.convert(stmt)
	if err != nil {
		return nil, nil, debug, err
	}

	var bodies []*bytes.Buffer

	/*
	 * Send indicators to get results back
	 */
	bodies, err = p.request(searchFields)
	var unpacked []string
	if err != nil {
		return nil, nil, debug, err
	}

	/*
	 * Receive hits and deserialize them
	 */

	// Attempt to unpack arrays if we get results from the bulk endpoint
	for _, body := range bodies {
		jsonParsed, err := gabs.ParseJSONBuffer(body)
		if err != nil {
			return nil, nil, debug, err
		}
		childM := jsonParsed.ChildrenMap()
		// If SHA-1 is present, this is not an array
		if _, ok := childM["SHA-1"]; ok {
			unpacked = append(unpacked, jsonParsed.String())
		} else {
			// Unpack the Array from bulk
			childA := jsonParsed.Children()
			for _, child := range childA {
				unpacked = append(unpacked, child.String())
			}
		}
	}

	// variable holding the Json values for the merge
	var finalJSONParsed = gabs.New()
	finalJSONParsed.Array("entries")
	// variable used to convert back to go
	var entries []map[string]interface{}

	for _, body := range unpacked {
		// Struct to store statistics data
		// when the amount of returned entries is too large
		stats := pdk.NewStats()

		for _, field := range p.source.StatsFields {
			stats.Fields[field] = sortedmap.New(10, desc.Int)
		}

		jsonParsed, err := gabs.ParseJSON([]byte(body))
		if err != nil {
			return nil, nil, debug, err
		}

		entryObj := gabs.New()
		entryObj.Array("entries")

		children := new(gabs.Container)
		parents := new(gabs.Container)

		// TODO fetch children / parents according to the LIMIT
		// if the children/parent key exist and the count hits hashlookup limits,
		// we query hashlookup again up the the query' LIMIT

		if jsonParsed.Exists("children") {
			if err != nil {
				return nil, nil, debug, err
			}
			// Here we run the new query
			gabsChildren, err := p.getRelated("children", jsonParsed.S("SHA-1").Data().(string), p.limit)
			if err != nil {
				return nil, nil, debug, err
			}
			children = gabsChildren.Path("children")
			for _, child := range children.Children() {
				// Create an object from the child
				jsonObj := gabs.New()
				// Add a reference to the parent
				jsonObj.Set(jsonParsed.S("SHA-1").Data(), "parent")
				jsonObj.Set(child.Data().(string), "SHA-1")
				// Add the children to the list of entries
				entryObj.ArrayAppend(jsonObj, "entries")
			}
			// Remove the children for the main received object
			jsonParsed.Delete("children")
		}
		if jsonParsed.Exists("parents") {
			// Here we run the new query
			gabsParents, err := p.getRelated("parents", jsonParsed.S("SHA-1").Data().(string), p.limit)
			if err != nil {
				return nil, nil, debug, err
			}
			parents = gabsParents.Path("parents")
			if err != nil {
				return nil, nil, debug, err
			}
			for _, child := range parents.Children() {
				// Create an object from the parent
				jsonObj := gabs.New()
				// Add a reference to the children
				jsonObj.Set(jsonParsed.S("SHA-1").Data(), "children")
				jsonObj.Set(child.Data().(string), "SHA-1")
				// Add the children to the list of entries
				entryObj.ArrayAppend(jsonObj, "entries")
			}
			// Remove the parents for the main received object
			jsonParsed.Delete("parents")
		}

		entryObj.ArrayAppend(jsonParsed, "entries")

		finalJSONParsed.Merge(entryObj)
	}

	err = json.NewDecoder(strings.NewReader(finalJSONParsed.S("entries").String())).Decode(&entries)
	if err != nil {
		return nil, nil, debug, err
	}

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

	for _, entry := range entries {

		// Stop when results count is over the limit
		if counter >= p.limit {
			// Uncomment in real plugin
			//cancel()

			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return nil, top, nil, nil
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

	return results, nil, debug, nil
}

// request connects to the HTTP access point and returns the response
func (p *plugin) request(searchFields [][2]string) ([]*bytes.Buffer, error) {
	// Create a request body
	data := url.Values{}

	// List of possible Hashlookup fields
	possibleFields := []string{"md5", "sha1", "sha256"}
	// List of requests we will have to make
	requests := make([]*http.Request, 0, 3)
	responses := make([]*bytes.Buffer, 0, 3)

	for _, field := range searchFields {
		data.Add(field[0], field[1])
	}

	for _, possibleSearchField := range possibleFields {
		// Check what endpoints we hit
		if data.Has(possibleSearchField) {
			tmpUrl := ""
			var tmpReq *http.Request
			var err error

			if len(data[possibleSearchField]) > 1 {
				tmpUrl = p.url + "/bulk/" + possibleSearchField
				tmpMap := make(map[string][]string)
				tmpMap["hashes"] = data[possibleSearchField]
				tmpBody, errm := json.Marshal(tmpMap)
				if errm != nil {
					return nil, fmt.Errorf("Could not serialize: %s", err.Error())
				}
				tmpReq, err = http.NewRequest("POST", tmpUrl, bytes.NewReader(tmpBody))
			} else {
				tmpUrl = p.url + "/lookup/" + possibleSearchField + "/" + data.Get(possibleSearchField)
				tmpReq, err = http.NewRequest("GET", tmpUrl, nil)
			}
			requests = append(requests, tmpReq)

			if err != nil {
				return nil, fmt.Errorf("Can't create the request: %s", err.Error())
			}
		}
	}
	// If no possible search field, exit
	if len(requests) == 0 {
		return nil, fmt.Errorf("%s", errors.New("No compatible hashlookup search field."))
	}

	for _, req := range requests {
		// Set basic auth credentials if given
		if p.source.Access["apiKey"] != "" {
			req.SetBasicAuth(p.source.Access["apiKey"], "")
		}
		req.Header.Add("Content-Type", "application/json")
		// Declare an HTTP client to execute the request
		client := http.Client{Timeout: p.source.Timeout}
		// Send an HTTP using `req` object
		tmpResp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Can't do an HTTP request: %s", err.Error())
		}

		tmpBody := &bytes.Buffer{}
		_, err = tmpBody.ReadFrom(tmpResp.Body)
		if err != nil {
			tmpResp.Body.Close()
			return nil, fmt.Errorf("Can't read an HTTP response: %s", err.Error())
		}
		tmpResp.Body.Close()
		// Check the response
		if tmpResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Bad response StatusCode: %s", tmpResp.Status)
		}
		responses = append(responses, tmpBody)
	}

	return responses, nil
}

// getChildren connect to the HTTP endpoints and retrieve children
func (p *plugin) getRelated(relation string, parent string, limit int) (*gabs.Container, error) {
	fmt.Println(relation)

	// Do everything as float64 as shortcut because it's gabs's default
	var nbElements, total, spoon float64
	gabsBuffer := gabs.New()
	gabsBuffer.Array(relation)
	var err error

	// How many records we want at once
	spoon = 99
	url := ""
	cursor := "0"
	total = spoon

	for (nbElements < float64(limit)) && (nbElements < total) {
		fmt.Println(nbElements)
		url = p.url + "/" + relation + "/" + parent + "/" + fmt.Sprint(spoon) + "/" + cursor
		fmt.Println(url)
		body, err := p.query(url)
		if err != nil {
			return nil, fmt.Errorf("Can't query %s", err.Error())
		}
		tmp, err := gabs.ParseJSONBuffer(body)
		if err != nil {
			return nil, fmt.Errorf("Can't parse: %s", err.Error())
		}
		gabsBuffer.Merge(tmp)
		cursor = tmp.Path("cursor").Data().(string)
		total, _ = tmp.Path("total").Data().(float64)
		nbElements += spoon
	}

	return gabsBuffer, err
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

func (p *plugin) query(url string) (*bytes.Buffer, error) {

	req, err := http.NewRequest("GET", url, nil)

	// Set basic auth credentials if given
	if p.source.Access["apiKey"] != "" {
		req.SetBasicAuth(p.source.Access["apiKey"], "")
	}
	req.Header.Add("Content-Type", "application/json")
	// Declare an HTTP client to execute the request
	client := http.Client{Timeout: p.source.Timeout}
	// Send an HTTP using `req` object
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Can't do an HTTP request: %s", err.Error())
	}

	body := &bytes.Buffer{}
	_, err = body.ReadFrom(response.Body)
	if err != nil {
		response.Body.Close()
		return nil, fmt.Errorf("Can't read an HTTP response: %s", err.Error())
	}
	response.Body.Close()
	// Check the response
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bad response StatusCode: %s", response.Status)
	}

	return body, err
}

// New method for Web GUI field names autocomplete:
func (p *plugin) Fields() ([]string, error) {
        return p.source.QueryFields, nil
}
