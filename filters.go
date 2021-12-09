package main

import (
	"encoding/json"
)

/*
 * Structure to hold search filter's state
 */
type Filter struct {
	// When filter is enabled - related request will be launched
	// to retrieve the data when web page is loaded
	Enabled bool `json:"enabled" bson:"enabled"`

	// User's original query
	Query string `json:"query" bson:"query"`
}

/*
 * Handle 'filters' websocket command to save all current user's filters
 */
func (a *Account) filtersHandler(filters string) {
	var list []map[string]*Filter
	err := json.Unmarshal([]byte(filters), &list)
	if err != nil {
		a.send("error", "Can't parse filters data.", "Can't save filters!")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't unmarshal filters data: " + err.Error())
		return
	}

	err = a.update("filters", list)
	if err != nil {
		a.send("error", err.Error(), "Can't save filters!")
		return
	}

	log.Debug().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Filters are saved")
}
