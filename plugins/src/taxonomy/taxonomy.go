package main

import (
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

	return nil
}

func (p *plugin) Process(relations []map[string]interface{}) ([]map[string]interface{}, error) {

	for _, relation := range relations {
		for k, v := range p.processor.Data["taxonomy"].(map[string]interface{}) {
			for _, part := range []string{"from", "to", "edge"} {
				rp := relation[part]

				if rp != nil && (rp.(map[string]interface{})[p.processor.Data["field"].(string)] == k ||
					(rp.(map[string]interface{})["attributes"] != nil &&
						rp.(map[string]interface{})["attributes"].(map[string]interface{})[p.processor.Data["field"].(string)] == k)) {

					if p.processor.Data["group"] != nil &&
						rp.(map[string]interface{})["group"] != p.processor.Data["group"].(string) {
						continue
					}

					taxRelation := p.createRelation(v.(string), k,
						rp.(map[string]interface{})["group"].(string),
						rp.(map[string]interface{})["search"].(string),
					)

					relations = append(relations, taxRelation)
				}
			}
		}
	}

	return relations, nil
}

/*
 * Generate new graph relation to display taxonomy info as a new node
 */
func (p *plugin) createRelation(tid, fid, group, search string) map[string]interface{} {
	from := map[string]interface{}{
		"id":     fid,
		"group":  group,
		"search": search,
	}

	to := map[string]interface{}{
		"id":     tid,
		"group":  "taxonomy",
		"search": "taxonomy",
	}

	// Resulting graph relation to return
	result := make(map[string]interface{})

	// Put it together
	result["from"] = from
	result["to"] = to
	result["source"] = p.Conf().Name

	return result
}

func (p *plugin) Stop() error {
	return nil
}
