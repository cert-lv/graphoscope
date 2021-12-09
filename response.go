package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/olekukonko/tablewriter"
	"github.com/yukithm/json2csv"
)

/*
 * Structure that API returns as a search result
 */
type APIresponse struct {
	// Graph relations data
	Relations []map[string]interface{} `json:"relations,omitempty"`

	// Statistics (like Top 10)
	// when the amount of returned results exceeds the limit
	Stats map[string]interface{} `json:"stats,omitempty"`

	// If some data source returns an error this message will be shown.
	// Graph data or statistics will be returned as well
	Error string `json:"error,omitempty"`

	// Allow safe writing to the slice
	sync.RWMutex
}

/*
 * Send search results to the API user.
 * Receives user's IP, name, output format and SQL query
 */
func (a *APIresponse) send(w http.ResponseWriter, ip, username, format, sql string) {
	_, err := fmt.Fprint(w, a.format(format))
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Str("sql", sql).
			Msg("Can't send an API response: " + err.Error())
	}
}

/*
 * Format search output data.
 * Receives a requested format, JSON will be used by default
 */
func (a *APIresponse) format(f string) string {

	// Validate the format value
	if f != "" && f != "json" && f != "table" {
		log.Error().Msg("Unexpected API response format requested: '" + f + "', JSON used instead")
		a.Error = "Unexpected API response format: '" + f + "', JSON used instead. " + a.Error
	}

	// Format the content if necessary
	if f == "table" {
		output := ""

		if a.Error != "" {
			output += "Error: " + a.Error + "\n"

			if len(a.Stats) != 0 || len(a.Relations) != 0 {
				output += "\n"
			}
		}

		if len(a.Stats) != 0 {
			output += "\"" + a.Stats["source"].(string) + "\" has too many results. "

			csv, err := formatTo(a.Stats, "table")
			if err != nil {
				output += "Error: " + err.Error()
			} else {
				output += "Stats based on limited data:\n\n" + csv
			}
		}

		if len(a.Relations) != 0 {
			csv, err := formatTo(a.Relations, "table")
			if err != nil {
				output += "Error: " + err.Error()
			} else {
				if len(a.Stats) != 0 {
					output += "\n\n"
				}
				output += csv
			}
		}

		return output
	}

	// Return JSON by default
	output, err := formatTo(a, "json")
	if err != nil {
		return `{"error":"` + err.Error() + `"}`
	}

	return output
}

/*
 * Format the given single object
 */
func formatTo(data interface{}, format string) (string, error) {
	if format == "table" {
		// JSON to CSV
		// to get all the existing headers
		csvSTR, err := json2csv.JSON2CSV(data)
		if err != nil {
			return "", fmt.Errorf("Can't convert API response to CSV: " + err.Error())
		}

		output := ""
		buf := bytes.NewBufferString(output)
		wr := json2csv.NewCSVWriter(buf)
		wr.HeaderStyle = json2csv.DotNotationStyle

		err = wr.WriteCSV(csvSTR)
		if err != nil {
			return "", fmt.Errorf("Can't format API response to CSV: " + err.Error())
		}

		// Read csv values using csv.Reader.
		// Strings splitting by \n and "," is not enough as some fields
		// may contain them
		csvReader := csv.NewReader(strings.NewReader(buf.String()))
		rows, err := csvReader.ReadAll()
		if err != nil {
			return "", fmt.Errorf("Can't parse CSV: " + err.Error())
		}

		// Find system internal fields to be removed.
		// rows[0] is a headers row
		fromSearch := -1
		toSearch := -1

		for i, header := range rows[0] {
			if header == "from.search" {
				fromSearch = i
			}
			if header == "to.search" {
				toSearch = i
			}
		}

		// Switch indexes in case 'To' element
		// comes before the 'From' element,
		// so the fields removing function works in the expected way
		if toSearch < fromSearch {
			i := toSearch
			toSearch = fromSearch
			fromSearch = i
		}

		rows[0] = removeFields(rows[0], fromSearch, toSearch)

		// Clear CSV data from buffer to render a table
		buf.Reset()
		table := tablewriter.NewWriter(buf)
		table.SetHeader(rows[0])

		for i := 1; i < len(rows); i++ {
			row := rows[i]

			row = removeFields(row, fromSearch, toSearch)
			table.Append(row)
		}

		table.Render()

		return buf.String(), nil
	}

	// Return JSON by default
	b, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("Can't format API response to JSON: " + err.Error())
	}

	return string(b), nil
}

/*
 * Remove system internal fields from an output formatted as table
 */
func removeFields(slice []string, f, t int) []string {
	slice = append(slice[:f], slice[f+1:]...)
	return append(slice[:t-1], slice[t:]...)
}
