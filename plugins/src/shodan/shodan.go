package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/cert-lv/graphoscope/pdk"
	"github.com/ns3777k/go-shodan/v4/shodan"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
)

/*
 * Temp structs to replace package's native code
 * and fix JSON timestamp parsing error.
 *
 * Original files:
 *   - https://github.com/ns3777k/go-shodan/blob/master/shodan/shodan.go
 *   - https://github.com/ns3777k/go-shodan/blob/master/shodan/dns.go
 *   - https://github.com/ns3777k/go-shodan/blob/master/shodan/errors.go
 */

const (
	dnsPath        = "/dns/domain/%s"
	errNoInfoForIP = "No information available for that IP."
)

type DomainDNSInfo struct {
	Domain     string              `json:"domain"`
	Tags       []string            `json:"tags"`
	Data       []*SubdomainDNSInfo `json:"data"`
	Subdomains []string            `json:"subdomains"`
}

type SubdomainDNSInfo struct {
	Subdomain string `json:"subdomain"`
	Type      string `json:"type"`
	Value     string `json:"value"`
	LastSeen  string `json:"last_seen"`
}

/*
 * Check "pdk/plugin.go" for the built-in plugin functions description
 */

func (p *plugin) Conf() *pdk.Source {
	return p.source
}

func (p *plugin) Setup(source *pdk.Source, limit int) error {

	// Set access key
	os.Setenv("SHODAN_KEY", source.Access["key"])

	// Store settings
	p.source = source
	p.limit = limit

	var err error
	p.pages, err = strconv.Atoi(source.Access["pages"])
	if err != nil {
		return fmt.Errorf("Can't parse 'pages' as integer")
	}

	p.credits, err = strconv.Atoi(source.Access["credits"])
	if err != nil {
		return fmt.Errorf("Can't parse 'credits' as integer")
	}

	// Set possible variable type & searching fields
	for _, relation := range source.Relations {
		for _, types := range relation.From.VarTypes {
			types.RegexCompiled = regexp.MustCompile(types.Regex)
		}
	}

	// fmt.Printf("Shodan %s: %#v\n\n", source.Name, p)
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
	 * Send query
	 */
	response, debug, err := p.request(searchField)
	if err != nil {
		return nil, nil, debug, err
	}

	// Struct to store statistics data
	// when the amount of returned entries is too large
	stats := pdk.NewStats()

	for _, field := range p.source.StatsFields {
		stats.Fields[field] = sortedmap.New(10, desc.Int)
	}

	mx := &sync.Mutex{}
	unique := make(map[string]bool)
	counter := 0

	// Iterate through the results
	for _, entry := range response {

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

// request connects to the API access point and returns the response
func (p *plugin) request(searchField [2]string) ([]map[string]interface{}, map[string]interface{}, error) {

	// List of entries and debug info to return to the client
	results := []map[string]interface{}{}
	debug := make(map[string]interface{})

	// Shodan client
	client := shodan.NewEnvClient(nil)
	var err error

	if searchField[0] == "ip" {
		results, err = p.getServicesForHost(client, results, searchField[1])
		if err != nil {
			return nil, nil, err
		}

		results, err = p.getDNSReverse(client, results, searchField[1])
		if err != nil {
			return nil, nil, err
		}

	} else if searchField[0] == "domain" {
		results, err = p.getDomain(client, results, searchField[1])
		if err != nil {
			return nil, nil, err
		}

		results, err = p.getDNSResolve(client, results, searchField[1])
		if err != nil {
			return nil, nil, err
		}

		results, err = p.getHostsForQuery(client, results, searchField[1])
		if err != nil {
			return nil, nil, err
		}

	} else if searchField[0] == "query" {
		results, err = p.getHostsForQuery(client, results, searchField[1])
		if err != nil {
			return nil, nil, err
		}

	} else if searchField[0] == "vulnerability" {
		results, err = p.searchExploits(client, results, searchField[1])
		if err != nil {
			return nil, nil, err
		}

	} else {
		fmt.Println("Unexpected field requested:" + searchField[0])
	}

	// Add info about credits left
	results, err = p.getAPIInfo(client, results)
	if err != nil {
		return nil, nil, err
	}

	return results, debug, nil
}

// Return all services that have been found on the given host IP.
// Requires membership or higher to access
func (p *plugin) getServicesForHost(client *shodan.Client, results []map[string]interface{}, ip string) ([]map[string]interface{}, error) {
	host, err := client.GetServicesForHost(context.Background(), ip, nil)
	if err != nil {
		if err.Error() == errNoInfoForIP {
			return results, nil
		}

		return nil, err
	}

	// Response example:

	// &shodan.Host{
	// OS:              "",
	// Ports:           []int{8080, 2082, 2083, 2053, 2086, 2087, 80, 8880, 8443, 443},
	// IP:              net.IP{0x0, 0x0, 0x0, 0x0, ... 0xac, 0x43, 0xa9, 0xd},
	// ISP:             "Example, Inc.",
	// Hostnames:       []string{"example.org"},
	// Organization:    "Example, Inc.",
	// Vulnerabilities: []string{"CVE-2024-5458", "CVE-2024-4577"},
	// ASN:             "AS12345",
	// LastUpdate:      "2024-11-19T03:54:45.858781",
	// Data:            []*shodan.HostData{(*shodan.HostData)(0xc000276c00), (*shodan.HostData)(0xc000276d80), (*shodan.HostData)(0xc000276f00), (*shodan.HostData)(0xc000277080), (*shodan.HostData)(0xc000277200), (*shodan.HostData)(0xc000277380), (*shodan.HostData)(0xc000277500), (*shodan.HostData)(0xc000277680), (*shodan.HostData)(0xc000277800), (*shodan.HostData)(0xc000277980)},
	// HostLocation:    shodan.HostLocation{City:"San Francisco", RegionCode:"CA", AreaCode:0, Latitude:37.7621, Longitude:-122.3971, Country:"United States", CountryCode:"US", CountryCode3:"", Postal:"", DMA:0}
	// }

	// &shodan.HostData{
	//	Product:"",
	//	Hostnames:[]string{},
	//	Version:"",
	//	Title:"",
	//	SSL:(*shodan.HostSSL)(nil),
	//	IP:net.IP{0x0, 0x0, 0x0, 0x0, ... 0xac, 0x43, 0xa9, 0xd},
	//	OS:"",
	//	Organization:"Example, Inc.",
	//	ISP:"Example, Inc.",
	//	CPE:[]string(nil),
	//	Data:"HTTP/1.1 400 Bad Request\r\nServer: example\r\nDate: Thu, 31 Oct 2024 13:46:46 GMT\r\nContent-Type: text/html\r\nContent-Length: 655\r\nConnection: close\r\nCF-RAY: -\r\n\r\n",
	//	ASN:"AS12345",
	//	Port:2053,
	//	HTML:"",
	//	Banner:"",
	//	Link:"",
	//	Transport:"tcp",
	//	Domains:[]string{},
	//	Timestamp:"2024-10-31T13:46:46.680369",
	//	DeviceType:"",
	//	Location:(*shodan.HostLocation)(0xc000432f80),
	//	ShodanData:map[string]interface {}{"crawler":"8f9776facb65747441d1d26b112981f75def6d58", "id":"5442e8ef-6e72-0000-874d-7a26b4a05983", "module":"auto", "options":map[string]interface {}{}, "region":"na"},
	//	Opts:map[string]interface {}{}
	// }

	for _, port := range host.Ports {
		for _, data := range host.Data {
			if port == data.Port {
				result := map[string]interface{}{
					"ip":         ip,
					"port":       port,
					"product":    data.Product,
					"version":    data.Version,
					"title":      data.Title,
					"cpe":        strings.Join(data.CPE, ", "),
					"banner":     data.Banner,
					"transport":  data.Transport,
					"timestamp":  data.Timestamp,
					"deviceType": data.DeviceType,
					"data":       strings.TrimRight(data.Data, "\r\n"),
				}

				if data.SSL != nil {
					ssl, _ := json.Marshal(data.SSL)
					result["ssl"] = string(ssl)
				}

				if len(data.Opts) > 0 {
					optsString, _ := json.Marshal(data.Opts)
					result["opts"] = string(optsString)
				}

				for _, domain := range data.Domains {
					domainip := map[string]interface{}{
						"domain": domain,
						"ip":     ip,
						"port":   port,
					}

					results = append(results, domainip)
				}

				results = append(results, result)
			}
		}
	}

	for _, hostname := range host.Hostnames {
		result := map[string]interface{}{
			"ip":       ip,
			"hostname": hostname,
		}

		results = append(results, result)
	}

	for _, vulnerability := range host.Vulnerabilities {
		result := map[string]interface{}{
			"ip":            ip,
			"vulnerability": vulnerability,
		}

		results = append(results, result)
	}

	result := map[string]interface{}{
		"os":                       host.OS,
		"ip":                       host.IP.String(),
		"isp":                      host.ISP,
		"organization":             host.Organization,
		"asn":                      host.ASN,
		"lastUpdate":               host.LastUpdate,
		"hostLocationCity":         host.HostLocation.City,
		"hostLocationRegionCode":   host.HostLocation.RegionCode,
		"hostLocationAreaCode":     host.HostLocation.AreaCode,
		"hostLocationLatitude":     host.HostLocation.Latitude,
		"hostLocationLongitude":    host.HostLocation.Longitude,
		"hostLocationCountry":      host.HostLocation.Country,
		"hostLocationCountryCode":  host.HostLocation.CountryCode,
		"hostLocationCountryCode3": host.HostLocation.CountryCode3,
		"hostLocationPostal":       host.HostLocation.Postal,
		"hostLocationDMA":          host.HostLocation.DMA,
	}

	results = append(results, result)

	return results, nil
}

// GetHostsForQuery searches Shodan using the same query syntax as the website and
// use facets to get summary information for different properties.
// Requires membership or higher to access
func (p *plugin) getHostsForQuery(client *shodan.Client, results []map[string]interface{}, query string) ([]map[string]interface{}, error) {
	counter := 0

	for page := 1; page <= p.pages; page++ {
		hostQueryOptions := &shodan.HostQueryOptions{
			Query: query,
			// Facets: "country",
			// Minify bool   `url:"minify,omitempty"`
			Page: page,
		}

		hostMatch, err := client.GetHostsForQuery(context.Background(), hostQueryOptions)
		if err != nil {
			return nil, err
		}

		counter += len(hostMatch.Matches)

		// Response example:

		// &shodan.HostMatch{
		//	Total:   35826886,
		//	Facets:  map[string][]*shodan.Facet{"country":[]*shodan.Facet{(*shodan.Facet)(0xc00030a4b0), ... (*shodan.Facet)(0xc00030a510)}},
		//	Matches: []*shodan.HostData{(*shodan.HostData)(0xc000336900), ... (*shodan.HostData)(0xc0007d4000)}}

		// &shodan.HostData{
		//	Product:"",
		//	Hostnames:[]string{},
		//	Version:"",
		//	Title:"",
		//	SSL:(*shodan.HostSSL)(nil),
		//	IP:net.IP{0x0, 0x0, 0x0, 0x0, ... 0xac, 0x43, 0xa9, 0xd},
		//	OS:"",
		//	Organization:"Example, Inc.",
		//	ISP:"Example, Inc.",
		//	CPE:[]string(nil),
		//	Data:"HTTP/1.1 400 Bad Request\r\nServer: example\r\nDate: Thu, 31 Oct 2024 13:46:46 GMT\r\nContent-Type: text/html\r\nContent-Length: 655\r\nConnection: close\r\nCF-RAY: -\r\n\r\n",
		//	ASN:"AS12345",
		//	Port:2053,
		//	HTML:"",
		//	Banner:"",
		//	Link:"",
		//	Transport:"tcp",
		//	Domains:[]string{},
		//	Timestamp:"2024-10-31T13:46:46.680369",
		//	DeviceType:"",
		//	Location:(*shodan.HostLocation)(0xc000432f80),
		//	ShodanData:map[string]interface {}{"crawler":"8f9776facb65747441d1d26b112981f75def6d58", "id":"5442e8ef-6e72-0000-874d-7a26b4a05983", "module":"auto", "options":map[string]interface {}{}, "region":"na"},
		//	Opts:map[string]interface {}{}
		// }

		for _, match := range hostMatch.Matches {
			result := map[string]interface{}{
				"product":                  match.Product,
				"version":                  match.Version,
				"title":                    match.Title,
				"ip":                       match.IP.String(),
				"os":                       match.OS,
				"organization":             match.Organization,
				"isp":                      match.ISP,
				"cpe":                      strings.Join(match.CPE, ", "),
				"data":                     strings.TrimRight(match.Data, "\r\n"),
				"asn":                      match.ASN,
				"port":                     match.Port,
				"html":                     match.HTML,
				"banner":                   match.Banner,
				"link":                     match.Link,
				"transport":                match.Transport,
				"timestamp":                match.Timestamp,
				"deviceType":               match.DeviceType,
				"hostLocationCity":         match.Location.City,
				"hostLocationRegionCode":   match.Location.RegionCode,
				"hostLocationAreaCode":     match.Location.AreaCode,
				"hostLocationLatitude":     match.Location.Latitude,
				"hostLocationLongitude":    match.Location.Longitude,
				"hostLocationCountry":      match.Location.Country,
				"hostLocationCountryCode":  match.Location.CountryCode,
				"hostLocationCountryCode3": match.Location.CountryCode3,
				"hostLocationPostal":       match.Location.Postal,
				"hostLocationDMA":          match.Location.DMA,
				// "shodanData":
			}

			if match.SSL != nil {
				ssl, _ := json.Marshal(match.SSL)
				result["ssl"] = string(ssl)
			}

			if len(match.Opts) > 0 {
				optsString, _ := json.Marshal(match.Opts)
				result["opts"] = string(optsString)
			}

			for _, hostname := range match.Hostnames {
				hostip := map[string]interface{}{
					"hostname": hostname,
					"ip":       match.IP.String(),
				}

				results = append(results, hostip)
			}

			for _, domain := range match.Domains {
				domainip := map[string]interface{}{
					"domain": domain,
					"ip":     match.IP.String(),
					"port":   match.Port,
				}

				results = append(results, domainip)
			}

			results = append(results, result)
		}

		// Return if one of limits exceeded
		if counter > p.limit {
			return results, nil
		}
		if len(hostMatch.Matches) < 100 {
			return results, nil
		}
	}

	return results, nil
}

// Get all the subdomains and other DNS entries for the given domain. Uses 1 query credit per lookup.
// Requires membership or higher to access
func (p *plugin) getDomain(client *shodan.Client, results []map[string]interface{}, domain string) ([]map[string]interface{}, error) {
	var info DomainDNSInfo
	path := fmt.Sprintf(dnsPath, domain)
	req, err := client.NewRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	// Response example:

	// main.DomainDNSInfo{
	//     Domain:     "example.org",
	//     Tags:       []string{"dmarc"},
	//     Data:       []*main.SubdomainDNSInfo{(*main.SubdomainDNSInfo)(0xc000ddd900), ...
	//     Subdomains: []string{"*", "_dmarc", "abuse"}
	// }

	// &main.SubdomainDNSInfo{Subdomain:"abuse", Type:"A", Value:"1.1.1.1", LastSeen:"2024-11-20T06:54:32.091000"}
	// &main.SubdomainDNSInfo{Subdomain:"", Type:"MX", Value:"mx1.example.org", LastSeen:"2024-11-20T06:59:24.261000"}
	// &main.SubdomainDNSInfo{Subdomain:"", Type:"NS", Value:"ns1.example.org", LastSeen:"2024-11-20T06:59:11.453000"}
	// &main.SubdomainDNSInfo{Subdomain:"", Type:"SOA", Value:"ns1.example.org", LastSeen:"2024-11-20T07:02:36.150000"}
	// &main.SubdomainDNSInfo{Subdomain:"contacts", Type:"CNAME", Value:"contacts.example.org.cdn.example.net", LastSeen:"2024-11-20T06:54:31.537000"}
	// &main.SubdomainDNSInfo{Subdomain:"_dmarc", Type:"TXT", Value:"v=DMARC1; p=quarantine; sp=quarantine; ..

	if err := client.Do(context.Background(), req, &info); err != nil {
		return nil, err
	}

	for _, i := range info.Data {
		result := map[string]interface{}{
			"domain":   info.Domain,
			"tags":     strings.Join(info.Tags, ", "),
			"type":     i.Type,
			"lastSeen": i.LastSeen,
		}

		if i.Subdomain != "*" && i.Subdomain != "" {
			result["subdomain"] = i.Subdomain + "." + info.Domain
		}

		if i.Type == "A" {
			result["ip"] = i.Value
		} else if i.Type == "MX" || i.Type == "NS" || i.Type == "SOA" || i.Type == "CNAME" {
			result["subdomain"] = i.Value
		} else if i.Type == "TXT" {
			result["txt"] = i.Value
		}

		results = append(results, result)
	}

	return results, nil
}

// GetDNSResolve looks up the IP address for the provided list of hostnames.
// Requires membership or higher to access
func (p *plugin) getDNSResolve(client *shodan.Client, results []map[string]interface{}, domain string) ([]map[string]interface{}, error) {
	response, err := client.GetDNSResolve(context.Background(), []string{domain})
	if err != nil {
		return nil, err
	}

	// Response example:
	// [map[hostname:example.org ip:1.1.1.1]]

	for k, v := range response {
		result := map[string]interface{}{
			"hostname": k,
			"ip":       v.String(),
		}

		results = append(results, result)
	}

	return results, nil
}

// Look up the hostnames that have been defined for the given list of IP addresses.
// Requires membership or higher to access
func (p *plugin) getDNSReverse(client *shodan.Client, results []map[string]interface{}, ipstr string) ([]map[string]interface{}, error) {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil, fmt.Errorf("Invalid IP requested")
	}

	response, err := client.GetDNSReverse(context.Background(), []net.IP{ip})
	if err != nil {
		return nil, err
	}

	// Response example:
	// map[string]*[]string{"8.8.8.8":(*[]string)(0xc000fb80c0)}

	for k, vs := range response {
		if vs == nil {
			continue
		}

		for _, v := range *vs {
			result := map[string]interface{}{
				"hostname": v,
				"ip":       k,
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// SearchExploits searches across a variety of data sources for exploits and
// use facets to get summary information.
// Requires membership or higher to access
func (p *plugin) searchExploits(client *shodan.Client, results []map[string]interface{}, query string) ([]map[string]interface{}, error) {
	counter := 0

	for page := 1; page <= p.pages; page++ {
		exploitSearchOptions := &shodan.ExploitSearchOptions{
			Query: query,
			// Facets: "country",
			// Minify bool   `url:"minify,omitempty"`
			Page: page,
		}

		hostMatch, err := client.SearchExploits(context.Background(), exploitSearchOptions)
		if err != nil {
			return nil, err
		}

		counter += len(hostMatch.Matches)

		// Response example:

		// &shodan.Exploit{
		//	ID:"2019-1234",
		//	BID:[]int{123456},
		//	CVE:[]string{"CVE-2019-1234"},
		//	MSB:[]string{},
		//	OSVDB:[]interface {}{},
		//	Description:"** REJECT ** DO NOT USE THIS CANDIDATE NUMBER. ConsultIDs: CVE-2019-12345. Reason: This candidate is a duplicate of CVE-2019-12345. Notes: All CVE users should reference CVE-2019-12345 instead of this candidate. All references and descriptions in this candidate have been removed to prevent accidental usage.",
		//	Source:"CVE",
		//	Author:interface {}(nil),
		//	Code:"",
		//	Date:"",
		//	Platform:interface {}(nil),
		//	Port:0,
		//	Type:"",
		//	Privileged:false,
		//	Rank:"",
		//	Version:""
		// }

		for _, match := range hostMatch.Matches {

			// https://github.com/ns3777k/go-shodan/blob/master/shodan/exploits.go
			for _, bid := range match.BID {
				idbid := map[string]interface{}{
					"id":                   match.ID,
					"bid":                  bid,
					"description":          match.Description,
					"vulnerability_source": match.Source,
					"related_source":       "Bugtraq",
					"author":               match.Author,
					"code":                 match.Code,
					"date":                 match.Date,
					"platform":             match.Platform,
					"port":                 match.Port,
					"type":                 match.Type,
					"privileged":           match.Privileged,
					"rank":                 match.Rank,
					"version":              match.Version,
				}

				results = append(results, idbid)
			}

			for _, cve := range match.CVE {
				idcve := map[string]interface{}{
					"id":                   match.ID,
					"cve":                  cve,
					"description":          match.Description,
					"vulnerability_source": match.Source,
					"related_source":       "CVE",
					"author":               match.Author,
					"code":                 match.Code,
					"date":                 match.Date,
					"platform":             match.Platform,
					"port":                 match.Port,
					"type":                 match.Type,
					"privileged":           match.Privileged,
					"rank":                 match.Rank,
					"version":              match.Version,
				}

				results = append(results, idcve)
			}

			for _, msb := range match.MSB {
				idmsb := map[string]interface{}{
					"id":                   match.ID,
					"msb":                  msb,
					"description":          match.Description,
					"vulnerability_source": match.Source,
					"related_source":       "Microsoft Security Bulletin",
					"author":               match.Author,
					"code":                 match.Code,
					"date":                 match.Date,
					"platform":             match.Platform,
					"port":                 match.Port,
					"type":                 match.Type,
					"privileged":           match.Privileged,
					"rank":                 match.Rank,
					"version":              match.Version,
				}

				results = append(results, idmsb)
			}

			for _, osvdb := range match.OSVDB {
				idosvdb := map[string]interface{}{
					"id":                   match.ID,
					"osvdb":                osvdb,
					"description":          match.Description,
					"vulnerability_source": match.Source,
					"related_source":       "OSVDB",
					"author":               match.Author,
					"code":                 match.Code,
					"date":                 match.Date,
					"platform":             match.Platform,
					"port":                 match.Port,
					"type":                 match.Type,
					"privileged":           match.Privileged,
					"rank":                 match.Rank,
					"version":              match.Version,
				}

				results = append(results, idosvdb)
			}
		}

		// Return if one of limits exceeded
		if counter > p.limit {
			return results, nil
		}
		if len(hostMatch.Matches) < 100 {
			return results, nil
		}
	}

	return results, nil
}

// GetAPIInfo returns information about the API plan belonging to the given API key
func (p *plugin) getAPIInfo(client *shodan.Client, results []map[string]interface{}) ([]map[string]interface{}, error) {
	response, err := client.GetAPIInfo(context.Background())
	if err != nil {
		return nil, err
	}

	// Response example:
	// &shodan.APIInfo{
	//	Plan:         "edu",
	//	QueryCredits: 199973,
	//	ScanCredits:  65536,
	//	Telnet:       true,
	//	HTTPS:        true,
	//	Unlocked:     true,
	//	UnlockedLeft: 199973
	// }

	for _, result := range results {
		result["shodanCreditsLeft"] = fmt.Sprintf("%d / %d", response.QueryCredits, p.credits)
	}

	return results, nil
}

func (p *plugin) Stop() error {

	// No error to check, so return nil
	return nil
}
