package main

import (
	"testing"

	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Test attributes modifications
 */
func TestProcess(t *testing.T) {

	// Empty plugin's instance to test
	c := &plugin{}

	processor := &pdk.Processor{
		Data: map[string]interface{}{
			"group": "name",

			"modify": []interface{}{
				map[string]interface{}{
					"field":       "age",
					"regex":       "\\d*",
					"replacement": "***",
				},
			},
		},
	}

	err := c.Setup(processor)
	if err != nil {
		t.Errorf("Can't setup a modify plugin: %s", err.Error())
	}

	// Pairs of data source plugins responses and the expected processing results
	table := []struct {
		entry    map[string]interface{}
		modified map[string]interface{}
	}{
		// No modifications should be made, because group is different
		{map[string]interface{}{
			"from": map[string]interface{}{
				"id":     "John",
				"group":  "neighbor",
				"search": "name",
				"attributes": map[string]interface{}{
					"age": "25",
				},
			},
		}, map[string]interface{}{
			"from": map[string]interface{}{
				"id":     "John",
				"group":  "name",
				"search": "name",
				"attributes": map[string]interface{}{
					"age": "25",
				},
			},
		}},

		// Age should be anonymized
		{map[string]interface{}{
			"from": map[string]interface{}{
				"id":     "John",
				"group":  "name",
				"search": "name",
				"attributes": map[string]interface{}{
					"age": "25",
				},
			},
		}, map[string]interface{}{
			"from": map[string]interface{}{
				"id":     "John",
				"group":  "name",
				"search": "name",
				"attributes": map[string]interface{}{
					"age": "***",
				},
			},
		}},
	}

	for _, row := range table {
		result, err := c.Process([]map[string]interface{}{row.entry})
		if err != nil {
			t.Errorf("Can't process '%s': %s", row.entry, err.Error())
			continue
		}

		modified := row.modified["from"].(map[string]interface{})
		from := result[0]["from"].(map[string]interface{})

		if from["id"] != modified["id"] {
			t.Errorf("Invalid modification of ID in \"%v\": \"%v\", expected: \"%v\"",
				row.entry["from"], from, modified)
			break
		}

		for k, v := range from["attributes"].(map[string]interface{}) {
			if modified["attributes"].(map[string]interface{})[k] != v {
				t.Errorf("Invalid modification of \"%v\": \"%v\", expected: \"%v\"",
					row.entry["from"], from, modified)
				break
			}
		}
	}
}
