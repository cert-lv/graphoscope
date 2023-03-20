/*
 * Data processing definition.
 * For YAML files in "../processor" by default.
 *
 * Check "../definitions/processors/processor.yaml.example" for the fields description
 */

package pdk

import (
	"time"
)

type Processor struct {
	Name    string                 `yaml:"name"`
	Plugin  string                 `yaml:"plugin"`
	Timeout time.Duration          `yaml:"timeout"`
	Data    map[string]interface{} `yaml:"data"`
}
