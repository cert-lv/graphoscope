package pdk

import (
	"fmt"
	"sync"

	"github.com/umpc/go-sortedmap"
)

/*
 * Structure to contain statistics data
 * if some data source has too many entries to return
 * (above the preconfigured limit)
 */
type Stats struct {
	Fields map[string]*sortedmap.SortedMap
	mx     sync.Mutex
}

func NewStats() *Stats {
	return &Stats{
		Fields: make(map[string]*sortedmap.SortedMap),
	}
}

/*
 * Update statistics of the received entries from some data source.
 * When the amount of returned entries becomes too large
 * users will receive the statistics info instead of the graph relations data.
 *
 * Receives:
 *     entry - single entry from a data source
 *     key   - statistics chart field to update,
 *             one entry increases the value by 1
 */
func (s *Stats) Update(entry map[string]interface{}, key string) {
	// Skip if value is missing
	value := entry[key]
	if value == nil || fmt.Sprint(value) == "" {
		return
	}

	s.mx.Lock()

	if val, ok := s.Fields[key].Get(value); ok {
		s.Fields[key].Replace(value, val.(int)+1)
	} else {
		s.Fields[key].Insert(value, 1)
	}

	s.mx.Unlock()
}

/*
 * Convert sorted-map object to the native map,
 * converted to the JSON later,
 * so the Web GUI can draw interactive charts.
 *
 * Receives a data source name
 */
func (s *Stats) ToJSON(source string) (map[string]interface{}, error) {

	// Map to store Top 10 entries
	json := make(map[string]interface{})

	// Identifier of the source data belongs to
	json["source"] = source

	for k, v := range s.Fields {
		i := 1

		iterCh, err := v.IterCh()
		if err != nil && len(v.Keys()) != 0 {
			return nil, err

		} else if len(v.Keys()) != 0 {
			defer iterCh.Close()

			group := make(map[string]int)

			for rec := range iterCh.Records() {
				//fmt.Printf("%+v\n", rec)

				group[fmt.Sprint(rec.Key)] = rec.Val.(int)

				// We want Top 10 here and started from i == 1
				if i > 9 {
					break
				}

				i++
			}

			json[k] = group
		}
	}

	return json, nil
}
