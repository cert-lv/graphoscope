package pdk

import (
	"strings"
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
	if keys == nil || len(keys) == 0 {
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
		if s[key] != t[key] {
			return false
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
