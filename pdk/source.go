/*
 * Data source definition.
 * For YAML files in "../sources" by default.
 *
 * Check "../sources/source.yaml.example" for the fields description
 */

package pdk

import (
	"regexp"
	"time"
)

type Source struct {
	Name            string            `yaml:"name"`
	Label           string            `yaml:"label"`
	Icon            string            `yaml:"icon"`
	Plugin          string            `yaml:"plugin"`
	InGlobal        bool              `yaml:"inGlobal"`
	IncludeDatetime bool              `yaml:"includeDatetime"`
	SupportsSQL     bool              `yaml:"supportsSQL"`
	Timeout         time.Duration     `yaml:"timeout"`
	Access          map[string]string `yaml:"access"`
	QueryFields     []string          `yaml:"queryFields"`
	IncludeFields   []string          `yaml:"includeFields"`
	StatsFields     []string          `yaml:"statsFields"`
	ReplaceFields   map[string]string `yaml:"replaceFields"`
	Relations       []*Relation       `yaml:"relations"`
}

type Relation struct {
	From *Node `yaml:"from"`
	To   *Node `yaml:"to"`
	Edge *struct {
		Label      string   `yaml:"label"`
		Attributes []string `yaml:"attributes"`
	} `yaml:"edge"`
}

type Node struct {
	ID         string   `yaml:"id"`
	Group      string   `yaml:"group"`
	Search     string   `yaml:"search"`
	Attributes []string `yaml:"attributes"`
	VarTypes   []*struct {
		Regex         string         `yaml:"regex"`
		RegexCompiled *regexp.Regexp `yaml:"-"`
		Group         string         `yaml:"group"`
		Search        string         `yaml:"search"`
		Label         string         `yaml:"label"`
	} `yaml:"varTypes"`
}
