package main

import (
	"testing"

	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Test processing the data source's response
 */
func TestProcess(t *testing.T) {

	// Empty plugin's instance to test
	c := &plugin{}

	processor := &pdk.Processor{
		Data: map[string]interface{}{
			"field": "id",
			"group": "type",

			"taxonomy": map[string]interface{}{
				"brute-force": "intrusion-attempts",
			},
		},
	}

	c.Setup(processor)

	// Pairs of data source plugins responses and the expected processing results
	table := []struct {
		entry    map[string]interface{}
		inserted map[string]interface{}
	}{
		// New relation should be added because "id" == "brute-force"
		{map[string]interface{}{
			"from": map[string]interface{}{
				"id":     "brute-force",
				"group":  "type",
				"search": "type",
			},
		}, map[string]interface{}{
			"id":     "intrusion-attempts",
			"group":  "taxonomy",
			"search": "taxonomy",
		}},

		// New relation should be added because "attributes.id" == "brute-force"
		{map[string]interface{}{
			"from": map[string]interface{}{
				"id":     "malware",
				"group":  "type",
				"search": "type",
				"attributes": map[string]interface{}{
					"id": "brute-force",
				},
			},
		}, map[string]interface{}{
			"id":     "intrusion-attempts",
			"group":  "taxonomy",
			"search": "taxonomy",
		}},

		// New relation should NOT be added because "id" value is not mentioned in processor.Data["taxonomy"]
		{map[string]interface{}{
			"from": map[string]interface{}{
				"id":     "ddos",
				"group":  "type",
				"search": "type",
			},
		}, nil},

		// New relation should NOT be added because "group" value is not equal to processor.Data["group"]
		{map[string]interface{}{
			"from": map[string]interface{}{
				"id":     "brute-force",
				"group":  "address",
				"search": "type",
			},
		}, nil},
	}

	for _, row := range table {
		result, err := c.Process([]map[string]interface{}{row.entry})
		if err != nil {
			t.Errorf("Can't process '%s': %s", row.entry, err.Error())
			continue
		}

		if row.inserted == nil && len(result) > 1 {
			t.Errorf("Unwanted relation added for \"%s\": \"%s\"", row.entry["from"], result[1])

		} else if len(result) == 1 && row.inserted != nil {
			t.Errorf("No new relations added for \"%s\": \"%s\", expected: \"%s\"", row.entry["from"], result, row.inserted)

		} else if len(result) > 1 && row.inserted != nil {
			for k, v := range result[1]["to"].(map[string]interface{}) {
				if row.inserted[k] != v {
					t.Errorf("Invalid taxonomy added for \"%s\": \"%s\", expected: \"%s\"",
						row.entry["from"], result[1]["to"], row.inserted)
					break
				}
			}
		}
	}
}
