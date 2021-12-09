package pdk

import (
	"strings"
)

/*
 * CopyPresentValues goes through a list "keys" and copies "source" maps's values,
 * if such exist, to the "target".
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
func CopyPresentValues(source map[string]interface{}, target map[string]interface{}, keys []string) {
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

// StringSliceContains checks whether the slice contains the given string
func StringSliceContains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}

	return false
}

// IntSliceContains checks whether the slice contains the given integer
func IntSliceContains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}

	return false
}
