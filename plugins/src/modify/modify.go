package main

import (
	"fmt"
	"regexp"

	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Check "pdk/plugin.go" for the built-in plugin functions description
 */

func (p *plugin) Conf() *pdk.Processor {
	return p.processor
}

func (p *plugin) Setup(processor *pdk.Processor) error {

	// Store settings
	p.processor = processor

	// Convert string regexs into actual regexps
	if p.processor.Data["modify"] != nil {
		for i, entry := range p.processor.Data["modify"].([]interface{}) {
			p.processor.Data["modify"].([]interface{})[i].(map[string]interface{})["regex"] = regexp.MustCompile(entry.(map[string]interface{})["regex"].(string))
		}
	}

	return nil
}

func (p *plugin) Process(relations []map[string]interface{}) ([]map[string]interface{}, error) {

	if p.processor.Data["modify"] != nil {
		for _, relation := range relations {
			for _, m := range p.processor.Data["modify"].([]interface{}) {
				for _, part := range []string{"from", "to", "edge"} {
					rp := relation[part]

					if rp != nil {
						rpt := relation[part].(map[string]interface{})

						if p.processor.Data["group"] != nil && rpt["group"] != p.processor.Data["group"].(string) {
							continue
						}

						mt := m.(map[string]interface{})

						if mt["field"].(string) == "id" {
							rpt["id"] = mt["regex"].(*regexp.Regexp).ReplaceAllString(rpt["id"].(string), fmt.Sprint(mt["replacement"]))
						}

						if rpt["attributes"] != nil && rpt["attributes"].(map[string]interface{})[mt["field"].(string)] != nil {
							rpt["attributes"].(map[string]interface{})[mt["field"].(string)] = mt["regex"].(*regexp.Regexp).ReplaceAllString(rpt["attributes"].(map[string]interface{})[mt["field"].(string)].(string), fmt.Sprint(mt["replacement"]))
						}
					}
				}
			}
		}
	}

	return relations, nil
}

func (p *plugin) Stop() error {
	return nil
}
