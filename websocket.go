package main

import (
	"encoding/json"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// HTTP request -> Websocket connection
	// upgrader with the default options
	upgrader = websocket.Upgrader{}

	// Send pings to the client with this period
	pingPeriod = 60 * time.Second

	// Online users for the faster access
	online = make(map[string]*Account)
)

/*
 * Structure of a single Websocket message
 */
type Message struct {
	// Type of the message: error, results, etc.
	Type string `json:"type"`

	// SQL query, response, etc.
	Data string `json:"data"`

	// Possible additional data
	Extra string `json:"extra,omitempty"`
}

/*
 * Accept Websocket connections on '/ws'
 */
func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().Msg("Can't get IP for the Websocket connection: " + err.Error())
		return
	}

	// Check whether user is signed in
	username, err := sessions.exists(w, r)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg("Websocket handler: " + err.Error())
		return
	}

	// Get account from a database
	account, err := db.getAccount(username)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't get account to establish a Websocket connection: " + err.Error())
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't upgrade to the Websocket: " + err.Error())
		return
	}

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Msg("Websocket connection established")

	account.Session = &Session{
		IP:             ip,
		Websocket:      ws,
		Request:        r,
		ResponseWriter: w,
	}

	online[username] = account

	// Listen for the incoming Websocket messages in a loop
	go account.listen()
}

/*
 * Listen for the incoming Websocket messages
 */
func (a *Account) listen() {
	a.Session.Done = make(chan bool)
	defer func() {
		close(a.Session.Done)
		a.Session.Websocket.Close()
		a.Session.Websocket = nil
		delete(online, a.Username)
	}()

	go a.ping()

	for {
		_, bytes, err := a.Session.Websocket.ReadMessage()
		if err != nil {
			switch err.(type) {
			case *websocket.CloseError:
				log.Info().
					Str("ip", a.Session.IP).
					Str("username", a.Username).
					Msg("Websocket connection closed by client")
				return
			case *net.OpError:
				log.Info().
					Str("ip", a.Session.IP).
					Str("username", a.Username).
					Msg("Websocket is closed")
				return

			default:
				log.Error().
					Str("ip", a.Session.IP).
					Str("username", a.Username).
					Msgf("Unexpected Websocket message type received: %T", err)
			}

			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Can't read Websocket message: " + err.Error())
			return
		}

		// Unmarshal message
		var message *Message
		err = json.Unmarshal(bytes, &message)
		if err != nil {
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Can't unmarshal Websocket message: " + err.Error())
			continue
		}

		/*
		 * Process all kind of message types
		 */

		switch message.Type {
		case "sql":
			a.sqlHandler(message.Data)
		case "common":
			a.commonHandler(message.Data, message.Extra)
		case "notes":
			a.notesHandler(message.Data)
		case "notes-save":
			a.notesSaveHandler(message.Data, message.Extra)
		case "uuid":
			a.regenerateUUID()
		case "account-save":
			a.saveHandler(message.Data)
		case "account-delete":
			a.delete()
		case "settings":
			a.settingsHandler(message.Data)
		case "users":
			a.usersHandler(message.Data)
		case "reload-collectors":
			a.reloadCollectorsHandler()
		case "notifications":
			a.notificationsHandler()
		case "filters":
			a.filtersHandler(message.Data)
		case "dashboard-save":
			a.saveDashboardHandler(message.Data)
		case "dashboard-delete":
			a.delDashboardHandler(message.Data, message.Extra)
		case "options":
			a.optionsHandler(message.Data)
		case "upload-lists":
			a.getUploadLists()
		}

		// select {
		// case <-a.Session.Done:
		// 	log.Info().
		// 		Str("ip", a.IP).
		//		Str("username", a.Username).
		// 		Msg("Websocket closed")
		// 	return
		// default:
		// }

		// Hide confidencial information
		if message.Type == "account-save" {
			message.Data = ""
		}

		bytes, err = json.Marshal(message)
		if err != nil {
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Can't marshal modified Websocket message: " + err.Error())
		}

		log.Debug().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Websocket message received: " + string(bytes))

		// Nothing to update if own account was deleted
		if message.Type == "account-delete" {
			continue
		}

		// Update user's last active time
		err = a.update("lastActive", time.Now())
		if err != nil {
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Can't update account to set 'lastActive' time: " + err.Error())
			continue
		}
	}
}

/*
 * Process user's search query
 */
func (a *Account) sqlHandler(sql string) {

	// Find requested data source
	match := reSource.FindStringSubmatch(sql)
	if len(match) != 2 {
		a.send("error", "Requested data source missing", sql)

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Str("sql", sql).
			Msg("Requested data source missing")
		return
	}
	source := match[1]

	// Query data sources for a new data
	response := querySources(source, sql, a.Options.Debug, a.Username)

	// Get users initial query
	sql = reDatetimeLimit.ReplaceAllString(sql, "")

	// Send the formatted response back
	a.send("results", response.format("json"), sql)

	// Allow OS to take memory back
	debug.FreeOSMemory()
}

/*
 * Find selected nodes common neighbors.
 * Receives a list of "field='value'" of selected nodes
 * and a datetime range to search in
 */
func (a *Account) commonHandler(data string, datetime string) {
	var queries []string
	err := json.Unmarshal([]byte(data), &queries)
	if err != nil {
		a.send("error", "Can't parse selected nodes.", "Error!")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't unmarshal common queries: " + err.Error())
		return
	}

	// Regular API response to send back,
	// contains relations, stats and error
	response := &APIresponse{}
	// A list of unique neighbours to display on a Web GUI right panel
	response_neighbors := [][2]interface{}{}

	if len(queries) > 1 {
		nodes := []string{}
		results := [][]map[string]interface{}{}

		for _, query := range queries {
			field := strings.SplitN(query, "='", 2)
			nodes = append(nodes, field[1][:len(field[1])-1])

			// Query data sources for a new data
			result := querySources("global", "FROM global WHERE ("+query+") AND datetime BETWEEN "+datetime, a.Options.Debug, a.Username)
			results = append(results, result.Relations)

			if result.Error != "" {
				response.Error = result.Error
			}
			if len(result.Stats) != 0 {
				response.Stats = result.Stats
			}
		}

		// To find common neighbors of all the selected nodes
		// we need to compare with the first one node only
		firsts := results[0]

		for _, first := range firsts {
			firstFrom := first["from"].(map[string]interface{})["id"]
			firstTo := first["to"].(map[string]interface{})["id"]

			// Skip cases when selected FROM node is a neighbor of the TO node
			if a.queriesInclude(nodes, firstFrom, firstTo) {
				continue
			}

			for i := 1; i < len(results); i++ {
				result := results[i]

				for _, edge := range result {
					edgeFrom := edge["from"].(map[string]interface{})["id"]
					edgeTo := edge["to"].(map[string]interface{})["id"]

					// Skip cases when selected FROM node is a neighbor of the TO node
					if a.queriesInclude(nodes, edgeFrom, edgeTo) {
						continue
					}

					if edgeFrom == firstFrom || edgeFrom == firstTo ||
						edgeTo == firstFrom || edgeTo == firstTo {

						response.Relations = append(response.Relations, edge, first)
						includes := false

						for _, node := range nodes {
							if node == edgeFrom {
								// Skip dublicate entries
								if !a.commonNeighborsInclude(response_neighbors, edgeTo) {
									response_neighbors = append(response_neighbors, [2]interface{}{edge["to"].(map[string]interface{})["group"], edgeTo})
									includes = true
									break
								}
							}
						}

						if !includes {
							for _, node := range nodes {
								if node == edgeTo {
									// Skip dublicate entries
									if !a.commonNeighborsInclude(response_neighbors, edgeFrom) {
										response_neighbors = append(response_neighbors, [2]interface{}{edge["from"].(map[string]interface{})["group"], edgeFrom})
										break
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Send common nodes back
	b, _ := json.Marshal(response_neighbors)
	a.send("common", response.format("json"), string(b))
}

/*
 * In some cases one selected node is a direct neighbor of another node,
 * we shouldn't return any of them
 */
func (a *Account) queriesInclude(queries []string, from, to interface{}) bool {
	i := 0

	for _, query := range queries {
		if query == from || query == to {
			i += 1

			if i == 2 {
				return true
			}
		}
	}

	return false
}

/*
 * Helper function to return only unique neighbors for the right Web GUI panel
 */
func (a *Account) commonNeighborsInclude(values [][2]interface{}, field interface{}) bool {
	for _, value := range values {
		if value[1] == field {
			return true
		}
	}

	return false
}

/*
 * Handle 'notes' websocket command
 * to get users notes for the graph element by its ID/value
 */
func (a *Account) notesHandler(id string) {
	if id == "" {
		a.send("notes-error", "Notes for an empty element requested.", "Error!")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Notes for an empty element requested")
		return
	}

	notes, err := db.getNotes(id)
	if err != nil {
		a.send("notes-error", err.Error(), "Can't get notes!")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't get notes: " + err.Error())
		return
	}

	a.send("notes", notes, "")
}

/*
 * Handle 'notes-save' websocket command
 * to set users notes for the graph element by its ID/value
 */
func (a *Account) notesSaveHandler(id, notes string) {
	if id == "" {
		a.send("notes-error", "Can't save notes for an empty element.", "Error!")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't save notes for an empty element")
		return
	}

	err := db.setNotes(id, strings.TrimSpace(notes))
	if err != nil {
		a.send("notes-error", err.Error(), "Can't set notes!")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Str("id", id).
			Str("notes", notes).
			Msg("Can't set notes: " + err.Error())
		return
	}

	a.send("notes-set", "", "")

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Str("id", id).
		Str("notes", notes).
		Msg("Notes set")
}

/*
 * Send a Websocket message to the client.
 * Receives all the values for the Message structure
 */
func (a *Account) send(tp, data, extra string) {
	m := &Message{
		Type:  tp,
		Data:  data,
		Extra: extra,
	}

	bytes, err := json.Marshal(m)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't marshal Websocket message: " + err.Error())
		return
	}

	if a.Session != nil {
		// Connections support one concurrent reader and one concurrent writer
		a.Session.WebsocketMutex.Lock()
		defer a.Session.WebsocketMutex.Unlock()

		err = a.Session.Websocket.WriteMessage(websocket.TextMessage, bytes)
		if err != nil {
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Can't write to the Websocket: " + err.Error())
		}
	}
}

/*
 * Ping Websocket client from time to time
 * for the connection to stay alive
 */
func (a *Account) ping() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.send("ping", "", "")
		case <-a.Session.Done:
			return
		}
	}
}

/*
 * Broadcast message to the all online users
 */
func broadcast(tp, data, extra string) {
	m := &Message{
		Type:  tp,
		Data:  data,
		Extra: extra,
	}

	bytes, err := json.Marshal(m)
	if err != nil {
		log.Error().Msg("Can't marshal Websocket message to broadcast: " + err.Error())
		return
	}

	for _, account := range online {
		if account.Session != nil {
			// Connections support one concurrent reader and one concurrent writer
			account.Session.WebsocketMutex.Lock()

			err = account.Session.Websocket.WriteMessage(websocket.TextMessage, bytes)
			if err != nil {
				log.Error().
					Str("ip", account.Session.IP).
					Str("username", account.Username).
					Msg("Can't write to the Websocket: " + err.Error())
			}

			account.Session.WebsocketMutex.Unlock()
		}
	}
}
