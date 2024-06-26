package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
)

/*
 * Check "pdk/plugin.go" for the built-in plugin functions description
 */

func (p *plugin) Conf() *pdk.Source {
	return p.source
}

func (p *plugin) Setup(source *pdk.Source, limit int) error {

	// Validate necessary parameters
	if source.Access["protocol"] != "http" && source.Access["protocol"] != "https" {
		return fmt.Errorf("'access.protocol' must be 'http[s]'")
	}

	if source.Access["host"] == "" {
		return fmt.Errorf("'access.host' is not defined")
	}

	if source.Access["apiKey"] == "" {
		return fmt.Errorf("'access.apiKey' is not defined")
	}

	// Store settings
	p.source = source
	p.limit = limit
	p.protocol = source.Access["protocol"]
	p.host = source.Access["host"]
	p.apiKey = source.Access["apiKey"]
	p.types = make(map[string]bool)

	p.caCertPath = source.Access["caCertPath"]
	p.certPath = source.Access["certPath"]
	p.keyPath = source.Access["keyPath"]

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}

		for _, types := range relation.To.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("MISP %s: %#v\n\n", source.Name, p)
	return nil
}

func (p *plugin) Fields() ([]string, error) {
	return p.source.QueryFields, nil
}

func (p *plugin) Search(stmt *sqlparser.Select) ([]map[string]interface{}, map[string]interface{}, map[string]interface{}, error) {

	// Storage for the results to return
	results := []map[string]interface{}{}

	// Convert SQL statement
	searchField, err := p.convert(stmt)
	if err != nil {
		return nil, nil, nil, err
	}

	/*
	 * Send indicators to get results back
	 */
	entries, debug, err := p.request(searchField)
	if err != nil {
		return nil, nil, debug, err
	}

	//fmt.Printf("MISP response:\n%v\n", body)

	// Struct to store statistics data
	// when the amount of returned entries is too large
	stats := pdk.NewStats()

	for _, field := range p.source.StatsFields {
		stats.Fields[field] = sortedmap.New(10, desc.Int)
	}

	/*
	 * Receive hits and deserialize them
	 */

	mx := &sync.Mutex{}
	unique := make(map[string]bool)
	counter := 0

	// Process results
	for _, entry := range entries {

		// Stop when results count is too big
		if counter >= p.limit {
			top, err := stats.ToJSON(p.source.Name)
			if err != nil {
				return nil, nil, debug, err
			}

			return nil, top, debug, nil
		}

		// Update stats
		for _, field := range p.source.StatsFields {
			stats.Update(entry, field)
		}

		pdk.CreateRelations(p.source, entry, unique, &counter, mx, &results)
	}

	return results, nil, debug, nil
}

// request connects to the MISP instance and returns the response
func (p *plugin) request(searchField []string) ([]map[string]interface{}, map[string]interface{}, error) {

	// Debug info
	debug := make(map[string]interface{})
	var entries []map[string]interface{}

	// Load TLS certificates
	clientTLSCert, err := tls.LoadX509KeyPair(p.certPath, p.keyPath)
	if err != nil {
		return nil, debug, fmt.Errorf("Error loading certificate and key file: %s", err.Error())
	}

	// Configure the client to trust TLS server certs issued by a CA
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, debug, fmt.Errorf("Can't create SystemCertPool: %s", err.Error())
	}

	if caCertPEM, err := os.ReadFile(p.caCertPath); err != nil {
		return nil, debug, fmt.Errorf("Can't read cert CA file: %s", err.Error())
	} else if ok := certPool.AppendCertsFromPEM(caCertPEM); !ok {
		return nil, debug, fmt.Errorf("Invalid cert CA file")
	}

	tlsConfig := &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{clientTLSCert},
		MinVersion:   tls.VersionTLS12, // At least TLS v1.2 is recommended
	}
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{Transport: tr}

	con := MispCon{p.protocol, p.host, p.apiKey, client}
	var mq MispQuery

	// Search for event by ID/UUID
	if searchField[0] == "event" {
		if len(searchField) == 2 {
			mq = MispEventQuery{
				EventID: searchField[1],
			}
		} else {
			mq = MispEventQuery{
				EventID: searchField[1],
				From:    searchField[2],
				To:      searchField[3],
			}
		}

		// Search for attribute of any type
	} else if searchField[0] == "attribute" {
		if len(searchField) == 2 {
			mq = MispAttributeQuery{
				Value: searchField[1],
			}
		} else {
			mq = MispAttributeQuery{
				Value: searchField[1],
				From:  searchField[2],
				To:    searchField[3],
			}
		}

		// Search for attribute of specific type
	} else {
		if len(searchField) == 2 {
			mq = MispAttributeQuery{
				Type:  searchField[0],
				Value: searchField[1],
			}
		} else {
			mq = MispAttributeQuery{
				Type:  searchField[0],
				Value: searchField[1],
				From:  searchField[2],
				To:    searchField[3],
			}
		}
	}

	debug["query"] = fmt.Sprint(mq)

	mr, err := con.Search(mq)
	if err != nil {
		return nil, debug, fmt.Errorf("Failed to search: %s", err.Error())
	}

	if searchField[0] == "event" {
		for mri := range mr.Iter() {
			for _, a := range mri.(MispEvent).Attribute {
				entry := map[string]interface{}{
					"EventID":             mri.(MispEvent).ID,
					"EventLabel":          mri.(MispEvent).ID + ": " + mri.(MispEvent).Info,
					"OrgcID":              mri.(MispEvent).OrgcID,
					"OrgID":               mri.(MispEvent).OrgID,
					"Date":                mri.(MispEvent).Date,
					"ThreatLevelID":       mri.(MispEvent).ThreatLevelID,
					"Info":                mri.(MispEvent).Info,
					"Published":           mri.(MispEvent).Published,
					"EventUUID":           mri.(MispEvent).UUID,
					"AttributeCount":      mri.(MispEvent).AttributeCount,
					"Analysis":            mri.(MispEvent).Analysis,
					"EventDistribution":   mri.(MispEvent).Distribution,
					"ProposalEmailLock":   mri.(MispEvent).ProposalEmailLock,
					"Locked":              mri.(MispEvent).Locked,
					"EventSharingGroupID": mri.(MispEvent).SharingGroupID,
					"Org":                 mri.(MispEvent).Org.Name,
					"Orgc":                mri.(MispEvent).Orgc.Name,

					a.Type: a.Value,

					"ID":             a.ID,
					"UUID":           a.UUID,
					"SharingGroupID": a.SharingGroupID,
					"Distribution":   a.Distribution,
					"Category":       a.Category,
					"ToIDS":          a.ToIDS,
					"Deleted":        a.Deleted,
					"Comment":        a.Comment,
				}

				// Get tags
				eventTags := []string{}
				for _, tag := range mri.(MispEvent).Tag {
					eventTags = append(eventTags, tag.Name)
				}

				attributeTags := []string{}
				for _, tag := range a.Tag {
					attributeTags = append(attributeTags, tag.Name)
				}

				entry["EventTags"] = strings.Join(eventTags[:], ", ")
				entry["AttributeTags"] = strings.Join(attributeTags[:], ", ")

				// Convert timestamps to datetime
				eventStrTimestamp, err := strconv.ParseInt(mri.(MispEvent).StrTimestamp, 10, 64)
				if err != nil {
					return nil, debug, fmt.Errorf("Failed to parse event's StrTimestamp: %s", mri.(MispEvent).StrTimestamp)
				}

				strPublishedTimestamp, err := strconv.ParseInt(mri.(MispEvent).StrPublishedTimestamp, 10, 64)
				if err != nil {
					return nil, debug, fmt.Errorf("Failed to parse event's StrPublishedTimestamp: %s", mri.(MispEvent).StrPublishedTimestamp)
				}

				strTimestamp, err := strconv.ParseInt(a.StrTimestamp, 10, 64)
				if err != nil {
					return nil, debug, fmt.Errorf("Failed to parse attribute's StrTimestamp: %s", a.StrTimestamp)
				}

				entry["EventStrTimestamp"] = time.Unix(eventStrTimestamp, 0)
				entry["StrPublishedTimestamp"] = time.Unix(strPublishedTimestamp, 0)
				entry["StrTimestamp"] = time.Unix(strTimestamp, 0)

				entries = append(entries, entry)
				p.generateRelations(a.Type)
			}

			for _, o := range mri.(MispEvent).Object {
				entry := map[string]interface{}{
					"EventID":             mri.(MispEvent).ID,
					"EventLabel":          mri.(MispEvent).ID + ": " + mri.(MispEvent).Info,
					"OrgcID":              mri.(MispEvent).OrgcID,
					"OrgID":               mri.(MispEvent).OrgID,
					"Date":                mri.(MispEvent).Date,
					"ThreatLevelID":       mri.(MispEvent).ThreatLevelID,
					"Info":                mri.(MispEvent).Info,
					"Published":           mri.(MispEvent).Published,
					"EventUUID":           mri.(MispEvent).UUID,
					"AttributeCount":      mri.(MispEvent).AttributeCount,
					"Analysis":            mri.(MispEvent).Analysis,
					"EventDistribution":   mri.(MispEvent).Distribution,
					"ProposalEmailLock":   mri.(MispEvent).ProposalEmailLock,
					"Locked":              mri.(MispEvent).Locked,
					"EventSharingGroupID": mri.(MispEvent).SharingGroupID,
					"Org":                 mri.(MispEvent).Org.Name,
					"Orgc":                mri.(MispEvent).Orgc.Name,

					"Label":          o.Name + ": " + o.ID,
					"ID":             o.ID,
					"Name":           o.Name,
					"Description":    o.Description,
					"UUID":           o.UUID,
					"SharingGroupID": o.SharingGroupID,
					"Distribution":   o.Distribution,
					"Deleted":        o.Deleted,
					"Comment":        o.Comment,
					"FirstSeen":      o.FirstSeen,
					"LastSeen":       o.LastSeen,
				}

				// Get tags
				eventTags := []string{}
				for _, tag := range mri.(MispEvent).Tag {
					eventTags = append(eventTags, tag.Name)
				}

				entry["EventTags"] = strings.Join(eventTags[:], ", ")

				// Convert timestamps to datetime
				eventStrTimestamp, err := strconv.ParseInt(mri.(MispEvent).StrTimestamp, 10, 64)
				if err != nil {
					return nil, debug, fmt.Errorf("Failed to parse event's StrTimestamp: %s", mri.(MispEvent).StrTimestamp)
				}

				strPublishedTimestamp, err := strconv.ParseInt(mri.(MispEvent).StrPublishedTimestamp, 10, 64)
				if err != nil {
					return nil, debug, fmt.Errorf("Failed to parse event's StrPublishedTimestamp: %s", mri.(MispEvent).StrPublishedTimestamp)
				}

				strTimestamp, err := strconv.ParseInt(o.StrTimestamp, 10, 64)
				if err != nil {
					return nil, debug, fmt.Errorf("Failed to parse objects's StrTimestamp: %s", o.StrTimestamp)
				}

				entry["EventStrTimestamp"] = time.Unix(eventStrTimestamp, 0)
				entry["StrPublishedTimestamp"] = time.Unix(strPublishedTimestamp, 0)
				entry["StrTimestamp"] = time.Unix(strTimestamp, 0)

				entries = append(entries, entry)
				p.generateRelations("Label")
			}
		}
	} else {
		for a := range mr.Iter() {
			entry := map[string]interface{}{
				a.(MispAttribute).Type: a.(MispAttribute).Value,

				"EventID":        a.(MispAttribute).EventID,
				"EventLabel":     a.(MispAttribute).EventID,
				"UUID":           a.(MispAttribute).UUID,
				"SharingGroupID": a.(MispAttribute).SharingGroupID,
				"Distribution":   a.(MispAttribute).Distribution,
				"Category":       a.(MispAttribute).Category,
				"ToIDS":          a.(MispAttribute).ToIDS,
				"Deleted":        a.(MispAttribute).Deleted,
				"Comment":        a.(MispAttribute).Comment,
			}

			// Get tags
			attributeTags := []string{}
			for _, tag := range a.(MispAttribute).Tag {
				attributeTags = append(attributeTags, tag.Name)
			}

			entry["AttributeTags"] = strings.Join(attributeTags[:], ", ")

			// Convert timestamps to datetime
			strTimestamp, err := strconv.ParseInt(a.(MispAttribute).StrTimestamp, 10, 64)
			if err != nil {
				return nil, debug, fmt.Errorf("Failed to parse attribute's StrTimestamp: %s", a.(MispAttribute).StrTimestamp)
			}

			entry["StrTimestamp"] = time.Unix(strTimestamp, 0)

			entries = append(entries, entry)
			p.generateRelations(a.(MispAttribute).Type)
		}
	}

	if err != nil {
		return nil, debug, fmt.Errorf("Failed to search: %s", err.Error())
	}

	return entries, debug, nil
}

func (p *plugin) generateRelations(typ string) {
	if _, ok := p.types[typ]; !ok {
		p.types[typ] = true

		relation := &pdk.Relation{
			From: &pdk.Node{
				ID:         typ,
				Group:      typ,
				Search:     typ,
				Attributes: []string{"ID", "Name", "Description", "UUID", "Comment", "Deleted", "Distribution", "SharingGroupID", "ToIDS", "FirstSeen", "LastSeen", "AttributeTags"},
			},
			To: &pdk.Node{
				ID:         "EventID",
				Group:      "event",
				Search:     "event",
				Attributes: []string{"EventID", "OrgcID", "OrgID", "Date", "ThreatLevelID", "Info", "Published", "EventUUID", "AttributeCount", "Analysis", "EventStrTimestamp", "EventDistribution", "ProposalEmailLock", "Locked", "StrPublishedTimestamp", "EventSharingGroupID", "Org", "Orgc", "EventTags"},
			},
			Edge: &struct {
				Label      string   `yaml:"label"`
				Attributes []string `yaml:"attributes"`
			}{
				Attributes: []string{"Category", "StrTimestamp"},
			},
		}

		// Search for objects is not supported
		if typ == "Label" {
			relation.From.Search = ""
		}

		// Set common nodes attributes for similar MISP field types
		if typ == "ip-src" || typ == "ip-dst" || typ == "domain|ip" {
			relation.From.Group = "ip"
			relation.From.Search = "ip"
		} else if typ == "domain" || typ == "hostname" {
			relation.From.Group = "domain"
			relation.From.Search = "domain"
		} else if typ == "email" || typ == "email-src" || typ == "email-dst" || typ == "target-email" || typ == "whois-registrant-email" || typ == "dns-soa-email" {
			relation.From.Group = "email"
			relation.From.Search = "email"
		}

		// Put backticks around uncommon field names
		if strings.Contains(relation.From.Search, "-") || strings.Contains(relation.From.Search, "|") {
			relation.From.Search = "`" + relation.From.Search + "`"
		}

		p.source.Relations = append(p.source.Relations, relation)
	}
}

func (p *plugin) Stop() error {

	// No error to check, so return nil
	return nil
}
