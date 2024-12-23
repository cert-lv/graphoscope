package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"

	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"golang.org/x/sync/errgroup"
)

var (
	// Regex to detect requested data source
	reSource = regexp.MustCompile(`(?i) *FROM +(.+?) +WHERE `)
)

/*
 * Serves '/api' to process API requests with an SQL query inside
 */
func apiHandler(w http.ResponseWriter, r *http.Request) {
	// Get requestor IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().Msg("User IP: " + r.RemoteAddr + " is not IP:port")
	}

	// User inputs:
	//   - auth UUID
	//   - output format
	//   - query debug info, disabled by default
	//   - SQL request
	uuid := r.FormValue("uuid")
	format := r.FormValue("format")
	showLimited := false
	includeDebug := false
	sql := r.FormValue("sql")

	// Response to send back
	response := &APIresponse{
		Relations: []map[string]interface{}{},
		Stats:     make(map[string]interface{}),
	}

	// Authenticate user
	account, err := db.getAccountByUUID(uuid)
	if err != nil {
		response.Error = "Can't authenticate user by the given UUID"
		response.send(w, ip, "", format, sql)

		log.Error().
			Str("ip", ip).
			Msg("Can't authenticate user by the given UUID: " + err.Error())
		return
	}

	// Validate SQL query
	if sql == "" {
		response.Error = "Query can't be empty"
		response.send(w, ip, account.Username, format, "")

		log.Error().
			Str("ip", ip).
			Str("username", account.Username).
			Msg("Query can't empty")
		return
	}

	// Show partial results when limit exceeded
	if r.FormValue("show_limited") == "true" {
		showLimited = true
	}

	// Disable debug info by default
	if r.FormValue("debug") == "true" {
		includeDebug = true
	}

	// Find requested data source
	match := reSource.FindStringSubmatch(sql)
	if len(match) != 2 {
		response.Error = "Requested data source missing"
		response.send(w, ip, account.Username, format, sql)

		log.Error().
			Str("ip", ip).
			Str("username", account.Username).
			Str("sql", sql).
			Msg("Requested data source missing")
		return
	}
	source := match[1]

	// Query data sources for the new relations
	response = querySources(source, sql, showLimited, includeDebug, account.Username)

	if len(response.Stats) != 0 {
		if response.Error != "" {
			response.Error += ". "
		}
		response.Error += "The amount of data has exceeded the limit"
	}

	response.send(w, ip, account.Username, format, sql)

	// Allow OS to take memory back
	debug.FreeOSMemory()
}

/*
 * Query all the requested data sources
 */
func querySources(source, sql string, showLimited, includeDebug bool, username string) *APIresponse {

	// Response to send back
	response := &APIresponse{
		Relations: []map[string]interface{}{},
		Stats:     make(map[string]interface{}),
		Debug:     make(map[string]interface{}),
	}

	// Check cache first
	if config.Database.CacheTTL != 0 {
		cache, err := db.getCache(sql)
		if err != nil {
			response.Error = "Can't query cache: " + err.Error()
		}

		if cache != nil {
			log.Info().
				Str("username", username).
				Str("sql", sql).
				Msg("Query from cache")

			response.Relations = cache.Relations
			response.Stats = cache.Stats

			return response
		}
	}

	// Group of concurrent queries to improve performance
	group, _ := errgroup.WithContext(context.Background())

	/*
	 * Use one specific collector
	 */

	if collector, ok := collectors[source]; ok {

		// Parse textual SQL into a syntax tree object
		queries, err := parseSQL(sql, collector.Conf().IncludeDatetime, collector.Conf().IncludeFields, collector.Conf().ReplaceFields, collector.Conf().SupportsSQL)
		if err != nil {
			response.Error = err.Error()

		} else {
			for i := range queries {
				// Additional variable to prevent "govet" tool's warning:
				// loopclosure: loop variable query captured by func literal
				query := queries[i]

				log.Info().
					Str("username", username).
					Str("sql", sql).
					Str("modified", sqlparser.String(query)).
					Msg("New request")

				// Run the search
				group.Go(func() error {
					result, stat, debug, err := collector.Search(query)
					if err != nil {
						return fmt.Errorf("%s", err.Error())
					}

					if stat != nil {
						response.Stats = stat
					}

					response.Lock()

					if stat == nil || (stat != nil && showLimited) {
						response.Relations = append(response.Relations, result...)
					}

					if includeDebug {
						response.Debug[collector.Conf().Name] = debug
					}

					response.Unlock()

					return nil
				})
			}
		}

		/*
		 * Search through the all collectors
		 */

	} else if source == "global" {

		// Use this pattern instead of 'for _, collector := range collectors {'
		// because Golang uses a pointer to the same collector
		// in every 'group.Go(func()', but we need to call everyone
		for key := range collectors {
			collector := collectors[key]

			// Skip some collectors,
			// for example very slow or without full featured query possibilities
			if !collector.Conf().InGlobal {
				continue
			}

			// Parse textual SQL into syntax tree object
			queries, err := parseSQL(sql, collector.Conf().IncludeDatetime, collector.Conf().IncludeFields, collector.Conf().ReplaceFields, collector.Conf().SupportsSQL)
			if err != nil {
				response.Error = err.Error()

			} else {
				for i := range queries {
					// Additional variable to prevent "govet" tool's warning:
					// loopclosure: loop variable query captured by func literal
					query := queries[i]

					log.Info().
						Str("username", username).
						Str("sql", sql).
						Str("modified", sqlparser.String(query)).
						Str("source", collector.Conf().Name).
						Msg("New global request")

					// Run the search
					group.Go(func() error {
						result, stat, debug, err := collector.Search(query)
						if err != nil {
							return fmt.Errorf("%s - %s", collector.Conf().Name, err.Error())
						}

						if stat != nil {
							response.Stats = stat
						}

						response.Lock()
						response.Relations = append(response.Relations, result...)

						if includeDebug {
							response.Debug[collector.Conf().Name] = debug
						}

						response.Unlock()

						return nil
					})
				}
			}
		}

		/*
		 * Unknown collector requested
		 */

	} else {
		response.Error = "Unknown data source requested"
	}

	// Check whether any goroutines failed
	err := group.Wait()
	if err != nil {
		response.Error = err.Error()
	}

	// Format warning for the Web GUI modal window,
	// but do not log styling to the file
	if response.Error != "" {
		log.Error().
			Str("username", username).
			Str("sql", sql).
			Msg("Search error: " + response.Error)

		response.Error = fmt.Sprintf("\"%s\" error: %s", source, response.Error)
	}

	if len(response.Relations) != 0 || len(response.Stats) != 0 {
		log.Debug().
			Str("username", username).
			Str("sql", sql).
			Interface("relations", response.Relations).
			Interface("stats", response.Stats).
			Msg("Data sent to the client")
	} else {
		log.Debug().
			Str("username", username).
			Str("sql", sql).
			Msg("No relations data found")
	}

	// Cache results to make the identical future requests faster
	if config.Database.CacheTTL != 0 {
		db.setCache(sql, response.Relations, response.Stats)
	}

	// Process received data by the processor plugins
	for _, processor := range processors {
		response.Relations, err = processor.Process(response.Relations)
		if err != nil {
			response.Error = fmt.Sprintf("\"%s\" error: %s", processor.Conf().Name, err.Error())
		}
	}

	// Return the request results
	return response
}
