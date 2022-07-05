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
	//   - SQL request
	uuid := r.FormValue("uuid")
	format := r.FormValue("format")
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
	response = querySources(source, sql, account.Username)
	response.send(w, ip, account.Username, format, sql)

	// Allow OS to take memory back
	debug.FreeOSMemory()
}

/*
 * Query all the requested data sources
 */
func querySources(source, sql, username string) *APIresponse {

	// Response to send back
	response := &APIresponse{
		Relations: []map[string]interface{}{},
		Stats:     make(map[string]interface{}),
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
		queries, err := parseSQL(sql, collector.Source().IncludeDatetime, collector.Source().IncludeFields, collector.Source().ReplaceFields, collector.Source().SupportsSQL)
		if err != nil {
			response.Error = err.Error()

		} else {
			for _, query := range queries {
				log.Info().
					Str("username", username).
					Str("sql", sql).
					Str("modified", sqlparser.String(query)).
					Msg("New request")

				// Run the search
				group.Go(func() error {
					result, stat, err := collector.Search(query)
					if err != nil {
						return fmt.Errorf("%s", err.Error())
					}

					if stat != nil {
						response.Stats = stat
					}

					response.Lock()
					response.Relations = append(response.Relations, result...)
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
			if !collector.Source().InGlobal {
				continue
			}

			// Parse textual SQL into syntax tree object
			queries, err := parseSQL(sql, collector.Source().IncludeDatetime, collector.Source().IncludeFields, collector.Source().ReplaceFields, collector.Source().SupportsSQL)
			if err != nil {
				response.Error = err.Error()

			} else {
				for _, query := range queries {
					log.Info().
						Str("username", username).
						Str("sql", sql).
						Str("modified", sqlparser.String(query)).
						Str("source", collector.Source().Name).
						Msg("New global request")

					group.Go(func() error {
						result, stat, err := collector.Search(query)
						if err != nil {
							return fmt.Errorf("%s - %s", collector.Source().Name, err.Error())
						}

						if stat != nil {
							response.Stats = stat
							//return nil
						}

						response.Lock()
						response.Relations = append(response.Relations, result...)
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
	if err := group.Wait(); err != nil {
		response.Error = err.Error()
	}

	// Format warning for the Web GUI modal window,
	// but do not log styling to the file
	if response.Error != "" {
		log.Error().
			Str("username", username).
			Str("sql", sql).
			Msg("Search error: " + response.Error)

		response.Error = fmt.Sprintf("<span class=\"red_fg\">\"%s\" error</span>: %s", source, response.Error)
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

	// Return the request results
	return response
}
