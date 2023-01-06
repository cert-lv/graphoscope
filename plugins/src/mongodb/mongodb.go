package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
 * Check "pdk/plugin.go" for the built-in plugin functions description
 */

func (p *plugin) Source() *pdk.Source {
	return p.source
}

func (p *plugin) Setup(source *pdk.Source, limit int) error {

	// Validate necessary parameters
	if source.Access["addr"] == "" {
		return fmt.Errorf("'access.addr' is not defined")
	} else if source.Access["db"] == "" {
		return fmt.Errorf("'access.db' is not defined")
	} else if source.Access["collection"] == "" {
		return fmt.Errorf("'access.collection' is not defined")
	}

	// MongoDB server address
	clientOptions := options.Client().ApplyURI("mongodb://" + source.Access["addr"])

	// Set credentials if given
	if source.Access["user"] != "" && source.Access["password"] != "" {
		credential := options.Credential{
			AuthSource: source.Access["db"],
			Username:   source.Access["user"],
			Password:   source.Access["password"],
		}

		clientOptions.SetAuth(credential)
	}

	// Be able to cancel execution when
	// some DB wants to return > limit amount of entries or time expires
	ctx, cancel := context.WithTimeout(context.Background(), source.Timeout)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}

	// Store settings
	p.source = source
	p.client = client
	p.limit = limit
	p.collection = client.Database(source.Access["db"]).Collection(source.Access["collection"])

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}

		for _, types := range relation.To.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("MongoDB %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {

	// First check for manually provided fields
	if len(p.source.QueryFields) != 0 {
		return p.source.QueryFields, nil
	}

	// Map for collecting unique fields only
	fieldsMap := make(map[string]bool)

	opts := options.Find().SetLimit(1000)
	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	cursor, err := p.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// Iterate through the results/cursor
	for cursor.Next(ctx) {
		// Deserialize
		entry := make(map[string]interface{})

		err := cursor.Decode(&entry)
		if err != nil {
			return nil, err
		}

		for field, data := range entry {
			switch data.(type) {
			case map[string]interface{}:
				p.getFields(data.(map[string]interface{}), field, fieldsMap)
			default:
				fieldsMap[field] = true
			}
		}
	}

	// Convert map to the slice
	fields := make([]string, 0, len(fieldsMap))
	for value, _ := range fieldsMap {
		fields = append(fields, value)
	}

	return fields, nil
}

func (p *plugin) getFields(data map[string]interface{}, field string, fields map[string]bool) {
	for f, m := range data {
		switch m.(type) {
		case map[string]interface{}:
			p.getFields(m.(map[string]interface{}), field+"."+f, fields)
		default:
			fields[field+"."+f] = true
		}
	}
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	filter, opts, err := p.convert(stmt, p.source.IncludeFields)
	if err != nil {
		return nil, nil, nil, err
	}

	// Debug info
	debug := make(map[string]interface{})

	filterBase64, err := json.Marshal(filter)
	if err != nil {
		return nil, nil, nil, err
	}
	optsBase64, err := json.Marshal(opts)
	if err != nil {
		return nil, nil, nil, err
	}

	debug["filter"] = string(filterBase64)
	debug["options"] = string(optsBase64)

	/*
	 * Run the query
	 */

	// Context to be able to cancel goroutines
	// when some DB wants to return > limit amount of entries
	// or time expires
	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	cursor, err := p.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, nil, debug, err
	}
	defer cursor.Close(ctx)

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

	// Iterate through the results/cursor
	for cursor.Next(ctx) {

		// Stop when results count is too big
		if counter >= p.limit {
			cancel()

			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return nil, top, debug, nil
		}

		// Deserialize
		entry := make(map[string]interface{})

		err := cursor.Decode(&entry)
		if err != nil {
			return nil, nil, debug, err
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
						pdk.CopyPresentValues(entry, result["edge"].(map[string]interface{})["attributes"].(map[string]interface{}), relation.Edge.Attributes)
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

	if err := cursor.Err(); err != nil {
		return nil, nil, debug, err
	}

	return results, nil, debug, nil
}

func (p *plugin) Stop() error {
	if p.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.source.Timeout)
	defer cancel()

	return p.client.Disconnect(ctx)
}
