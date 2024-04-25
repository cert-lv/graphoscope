package pdk

import (
	"fmt"
	"strings"
	"sync"
)

/*
 * Go through the list "keys" and copy "source" maps's values,
 * if such exist, into the "target".
 *
 * To be able to copy a field value of the internal child maps -
 * it's possible to specify a dot in a key name, function will split the
 * name by dots and try to find the expected internal map and it's field value.
 *
 * Example:
 *
 * The source: {
 *                 "x": 10,
 *                 "y": {
 *                     "z": "hello"
 *                 },
 *                 "n.m": "world"
 *             }
 *
 * Keys to copy:
 *
 *   x   - will copy the value "10"
 *   y.z - will copy the value "hello"
 *   n.m - will copy the value "world", as there is no internal map "n"
 */
func CopyPresentValues(source, target map[string]interface{}, keys []string) {
	for _, key := range keys {
		// If the whole string is a key - just assign a value
		if source[key] != nil {
			target[key] = source[key]

		} else if strings.Contains(key, ".") {
			right := source
			prefix := ""

			for {
				if strings.Contains(key, ".") {
					s := strings.Split(key, ".")

					if v, ok := right[s[0]].(map[string]interface{}); ok {
						right = v
					}

					prefix += s[0] + "."
					key = strings.Join(s[1:], ".")

					// When no more dots in key name - assign
					// the map's value to the target
				} else {
					if right[key] != nil {
						target[prefix+key] = right[key]
					}
					break
				}
			}
		}
	}
}

/*
 * Check whether the same nodes (by their ID) from different data source entries
 * contain identical attributes
 */
func attributesAreIdentical(source, target interface{}, keys []string) bool {
	if len(keys) == 0 {
		return true
	}

	if source == nil && target == nil {
		return true
	} else if source == nil && target != nil {
		return false
	} else if source != nil && target == nil {
		return false
	}

	s := source.(map[string]interface{})
	t := target.(map[string]interface{})

	for _, key := range keys {
		// If target doesn't contain such attribute -
		// return node to the client in any case
		if t[key] == nil {
			return false
		}

		// Otherwise compare all the values
		switch sts := s[key].(type) {
		case []interface{}:
			switch tts := t[key].(type) {
			case []interface{}:
				if len(sts) != len(tts) {
					return false
				}

				for _, st := range sts {
					if !InterfaceSliceContains(tts, st) {
						return false
					}
				}

			default:
				if len(sts) == 1 && fmt.Sprintf("%v", sts[0]) == fmt.Sprintf("%v", tts) {
					return true
				}
				return false
			}

		default:
			if s[key] != t[key] {
				return false
			}
		}
	}

	return true
}

/*
 * Check whether current entry is identical to any of
 * the already collected unique results.
 *
 * This returns to the user only unique entries from a data source.
 * If single relation's From.ID == To.ID, but even one attribute is different -
 * JavaScript-side will get both entries and merge their attributes
 */
func ResultsContain(results []map[string]interface{}, entry map[string]interface{}, relation *Relation) bool {
	for _, result := range results {
		from := result["from"].(map[string]interface{})
		to := result["to"].(map[string]interface{})

		if from["id"] == entry[relation.From.ID] &&
			to["id"] == entry[relation.To.ID] {

			// Compare attributes of FROM and TO nodes and EDGE
			if attributesAreIdentical(entry, from["attributes"], relation.From.Attributes) &&
				attributesAreIdentical(entry, to["attributes"], relation.To.Attributes) &&
				(relation.Edge == nil || attributesAreIdentical(entry, result["edge"], relation.Edge.Attributes)) {

				return true
			}
		}
	}

	return false
}

/*
 * Check whether the slice contains the given string
 */
func StringSliceContains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}

	return false
}

/*
 * Check whether the slice contains the given integer
 */
func IntSliceContains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}

	return false
}

/*
 * Check whether the slice contains the given interface
 */
func InterfaceSliceContains(slice []interface{}, val interface{}) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}

	return false
}

/*
 * Create relations from a data source response's single entry
 */
func CreateRelations(source *Source, entry map[string]interface{}, unique map[string]bool, counter *int, mx *sync.Mutex, results *[]map[string]interface{}) {
	// Go through all the predefined relations and collect unique entries
	for _, relation := range source.Relations {
		if entry[relation.From.ID] != nil && entry[relation.To.ID] != nil {
			mx.Lock()

			// Use "Printf(...%v..." instead of "entry[relation.From.ID].(string)"
			// as the value can be not a string only
			if _, exists := unique[fmt.Sprintf("%v-%v-%v-%v", relation.From.ID, entry[relation.From.ID], relation.To.ID, entry[relation.To.ID])]; exists {
				if ResultsContain(*results, entry, relation) {
					mx.Unlock()
					continue
				}
			}

			*counter++

			unique[fmt.Sprintf("%v-%v-%v-%v", relation.From.ID, entry[relation.From.ID], relation.To.ID, entry[relation.To.ID])] = true
			mx.Unlock()

			/*
			 * Check if expected relation exists in received data.
			 * This allows returned JSON objects to have dynamic schema
			 */
			if _, ok := entry[relation.From.ID]; !ok {
				continue
			}

			if _, ok := entry[relation.To.ID]; !ok {
				continue
			}

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
						from["label"] = t.Label

						break
					}
				}
			}

			if len(relation.From.Attributes) > 0 {
				from["attributes"] = make(map[string]interface{})
				CopyPresentValues(entry, from["attributes"].(map[string]interface{}), relation.From.Attributes)
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
						to["label"] = t.Label

						break
					}
				}
			}

			if len(relation.To.Attributes) > 0 {
				to["attributes"] = make(map[string]interface{})
				CopyPresentValues(entry, to["attributes"].(map[string]interface{}), relation.To.Attributes)
			}

			// Resulting graph entry to return
			result := make(map[string]interface{})

			/*
			 * Edge between FROM and TO
			 */
			if relation.Edge != nil && (relation.Edge.Label != "" || len(relation.Edge.Attributes) > 0) {
				result["edge"] = make(map[string]interface{})

				if to["label"] != "" && to["label"] != nil {
					result["edge"].(map[string]interface{})["label"] = to["label"]
				} else if from["label"] != "" && from["label"] != nil {
					result["edge"].(map[string]interface{})["label"] = from["label"]
				} else if relation.Edge.Label != "" {
					result["edge"].(map[string]interface{})["label"] = relation.Edge.Label
				}

				if len(relation.Edge.Attributes) > 0 {
					result["edge"].(map[string]interface{})["attributes"] = make(map[string]interface{})
					CopyPresentValues(entry, result["edge"].(map[string]interface{})["attributes"].(map[string]interface{}), relation.Edge.Attributes)
				}
			}

			/*
			 * Put it together
			 */
			result["from"] = from
			result["to"] = to
			result["source"] = source.Name

			//fmt.Println("Edge:", from, to, source)

			/*
			 * Add current entry to the list to return
			 */
			mx.Lock()
			*results = append(*results, result)
			mx.Unlock()
		}
	}
}
