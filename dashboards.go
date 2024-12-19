package main

import (
	"encoding/json"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
 * Structure to hold saved dashboard in a database
 */
type Dashboard struct {
	// Dashboard's unique name given by the user
	Name string `json:"name" bson:"_id"`

	// Who created this dashboard
	Owner string `json:"-" bson:"owner"`

	// A list of filters of this dashboard.
	// Will not be modified until dashboard is deleted/recreated
	Filters []map[string]*Filter `json:"filters" bson:"filters"`

	// Datetime range to search data in
	Datetime string `json:"datetime" bson:"datetime"`

	// Whether dashboard is visible to all users
	Shared bool `json:"shared" bson:"shared"`
}

/*
 * Handle 'dashboard-save' websocket command to save current dashboard.
 * Receives a JSON with all "Dashboard"'s fields filled
 */
func (a *Account) saveDashboardHandler(data string) {

	var dashboard *Dashboard
	err := json.Unmarshal([]byte(data), &dashboard)
	if err != nil {
		a.send("error", "Can't parse dashboard data.", "Can't save dashboard!")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't unmarshal dashboard data: " + err.Error())
		return
	}

	dashboard.Owner = a.Username

	// Skip empty name or filters
	if dashboard.Name == "" || (len(dashboard.Filters[0]) == 0 && len(dashboard.Filters[1]) == 0) {
		a.send("error", "Dashboard name and filters can't be empty.", "Can't save dashboard!")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Dashboard name and filters can't be empty")
		return
	}

	bytes, err := json.Marshal(dashboard)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't marshal websocket message: " + err.Error())
		return
	}

	/*
	 * Save as shared dashboard
	 */
	if dashboard.Shared {
		// Check whether already exists first
		existing := &Dashboard{}

		ctx, cancel := db.newContext()
		defer cancel()

		err = db.Dashboards.FindOne(ctx, bson.M{"_id": dashboard.Name}).Decode(existing)
		if err != nil && err != mongo.ErrNoDocuments {
			// Replace some characters as the error may contain:
			// got <invalid reflect.Value>
			// which shouldn't be rendered as HTML
			e := strings.Replace(strings.Replace(err.Error(), "<", "&lt;", -1), ">", "&gt;", -1)
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Can't check whether dashboard name is already reserved: " + e)

			a.send("error", "Can't check whether dashboard name is already reserved: "+e, "Can't save dashboard!")
			return
		}

		if err == mongo.ErrNoDocuments {
			// Save in a database
			_, err = db.Dashboards.InsertOne(ctx, dashboard)
			if err != nil {
				log.Error().
					Str("ip", a.Session.IP).
					Str("username", a.Username).
					Msg("Error while inserting shared dashboard: " + err.Error())

				a.send("error", err.Error(), "Can't save dashboard!")
				return
			}

			broadcast("dashboard-saved", string(bytes), "")

			log.Info().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Shared dashboard is saved: " + dashboard.Name)

		} else {
			a.send("error", "Shared dashboard name is already reserved.", "Can't save dashboard!")

			log.Info().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Shared dashboard name is already reserved: " + dashboard.Name)
		}

		/*
		 * Save as own/private dashboard
		 */
	} else {
		// Check whether already exists first
		_, exists := a.Dashboards[dashboard.Name]
		if exists {
			a.send("error", "Dashboard name is already reserved.", "Can't save dashboard!")

			log.Info().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Dashboard name is already reserved: " + dashboard.Name)

		} else {
			a.Dashboards[dashboard.Name] = dashboard

			// Save in a database
			err = a.update("dashboards", a.Dashboards)
			if err != nil {
				a.send("error", err.Error(), "Can't save dashboard!")
				return
			}

			a.send("dashboard-saved", string(bytes), "")

			log.Info().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msg("Dashboard is saved: " + dashboard.Name)
		}
	}
}

/*
 * Handle 'dashboard-delete' websocket command to delete selected dashboard
 * by its name. "shared" says to search in a private or shared lists
 */
func (a *Account) delDashboardHandler(name, shared string) {

	// Skip empty name or filters
	if name == "" {
		a.send("error", "Dashboard name can't be empty", "")

		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Dashboard name can't be empty to delete")
		return
	}

	/*
	 * Delete shared dashboard
	 */
	if shared == "true" {
		ctx, cancel := db.newContext()
		defer cancel()

		_, err := db.Dashboards.DeleteOne(ctx, bson.M{"_id": name})
		if err != nil {
			a.send("error", err.Error(), "")

			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msgf("Can't delete shared dashboard '%s': %s", name, err.Error())
			return
		}

		broadcast("dashboard-deleted", name, "true")

		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Shared dashboard is deleted: " + name)

		/*
		 * Delete private dashboard
		 */
	} else {
		delete(a.Dashboards, name)
		err := a.update("dashboards", a.Dashboards)
		if err != nil {
			a.send("error", err.Error(), "")

			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Msgf("Can't delete dashboard '%s': %s", name, err.Error())
			return
		}

		a.send("dashboard-deleted", name, "false")

		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Dashboard is deleted: " + name)
	}
}
