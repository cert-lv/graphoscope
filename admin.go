package main

import (
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Regex to validate Hex color definition
	reColor = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}){1,2}$`)
)

/*
 * Configurable graph UI settings.
 * Accessible to admins only at Web GUI settings -> "Administration".
 *
 * More details at:
 *   https://visjs.github.io/vis-network/docs/network/nodes.html
 *   https://visjs.github.io/vis-network/docs/network/edges.html
 *   https://visjs.github.io/vis-network/docs/network/interaction.html
 */
type GraphSettings struct {
	ID string `bson:"_id" json:"-"`

	NodeSize        int    `bson:"nodeSize"`
	BorderWidth     int    `bson:"borderWidth"`
	BGcolor         string `bson:"bgColor"`
	BorderColor     string `bson:"borderColor"`
	NodeFontSize    int    `bson:"nodeFontSize"`
	EdgeWidth       int    `bson:"edgeWidth"`
	EdgeColor       string `bson:"edgeColor"`
	EdgeFontSize    int    `bson:"edgeFontSize"`
	EdgeFontColor   string `bson:"edgeFontColor"`
	Shadow          bool   `bson:"shadow"`
	Arrow           bool   `bson:"arrow"`
	Smooth          bool   `bson:"smooth"`
	Hover           bool   `bson:"hover"`
	MultiSelect     bool   `bson:"multiSelect"`
	HideEdgesOnDrag bool   `bson:"hideEdgesOnDrag"`
}

/*
 * Serve '/admin' page
 */
func adminHandler(w http.ResponseWriter, r *http.Request) {
	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().Msg("Can't get IP for admin page: " + err.Error())
		return
	}

	// Check existing session
	username, err := sessions.exists(w, r)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg(err.Error())

		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Get account from DB
	account, err := db.getAccount(username)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't GetAccount for admin page: " + err.Error())

		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// All are admins in service's DEV mode
	if config.Environment != "prod" {
		account.Admin = true
	}

	// Drop non-admin users in prod. mode
	if !account.Admin && config.Environment == "prod" {
		log.Info().
			Str("ip", ip).
			Str("username", username).
			Msg("Not admin requested admin page")

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Error message to display in case it exists
	msg := ""

	settings, err := db.getGraphSettings()
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't get graph UI settings: " + err.Error())

		msg = err.Error()
	}

	accounts, err := db.getAccounts()
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't get accounts: " + err.Error())

		msg = err.Error()
	}

	templateData := &TemplateData{
		Account:       account,
		Accounts:      accounts,
		GraphSettings: settings,
		Error:         msg,
	}

	// Render HTML and send it to the client
	renderTemplate(w, "admin", templateData, nil)

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Msg("Admin page requested")
}

/*
 * Handle 'settings' websocket command to save global graph settings
 */
func (a *Account) settingsHandler(settings string) {

	// Drop non-admin users request
	if !a.Admin && config.Environment == "prod" {
		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Not enough rights to set UI settings")

		a.send("error", "Not enough rights to set UI settings.", "Can't save!")
		return
	}

	parts := strings.Split(settings, ",")

	// Get and validate all received settings
	nodesize, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse node size value: " + err.Error())

		a.send("error", "Invalid <strong>node size</strong> value given, integer expected.", "Can't save!")
		return
	}

	borderwidth, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse node border width value: " + err.Error())

		a.send("error", "Invalid <strong>node border width</strong> value given, integer expected.", "Can't save!")
		return
	}

	bgcolor := parts[2]
	if !reColor.MatchString(bgcolor) {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse node background color value: " + bgcolor)

		a.send("error", "Invalid <strong>node background color</strong> value given, <strong>#000</strong> or <strong>#000000</strong> format expected.", "Can't save!")
		return
	}

	bordercolor := parts[3]
	if !reColor.MatchString(bordercolor) {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse node border color value: " + bordercolor)

		a.send("error", "Invalid <strong>node border color</strong> value given, <strong>#000</strong> or <strong>#000000</strong> format expected.", "Can't save!")
		return
	}

	nodefontsize, err := strconv.Atoi(parts[4])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse node font size value: " + err.Error())

		a.send("error", "Invalid <strong>node font size</strong> value given, integer expected.", "Can't save!")
		return
	}

	shadow, err := strconv.ParseBool(parts[5])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse shadow value: " + err.Error())

		a.send("error", "Invalid <strong>node shadow</strong> value given, boolean expected.", "Can't save!")
		return
	}

	edgewidth, err := strconv.Atoi(parts[6])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse edge width value: " + err.Error())

		a.send("error", "Invalid <strong>edge width</strong> value given, integer expected.", "Can't save!")
		return
	}

	edgecolor := parts[7]
	if !reColor.MatchString(edgecolor) {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse edge color value: " + edgecolor)

		a.send("error", "Invalid <strong>edge color</strong> value given, <strong>#000</strong> or <strong>#000000</strong> format expected.", "Can't save!")
		return
	}

	edgefontsize, err := strconv.Atoi(parts[8])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse edge font size value: " + err.Error())

		a.send("error", "Invalid <strong>edge font size</strong> value given, integer expected.", "Can't save!")
		return
	}

	edgefontcolor := parts[9]
	if !reColor.MatchString(edgefontcolor) {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse edge font color value: " + edgefontcolor)

		a.send("error", "Invalid <strong>edge font color</strong> value given, <strong>#000</strong> or <strong>#000000</strong> format expected.", "Can't save!")
		return
	}

	arrow, err := strconv.ParseBool(parts[10])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse edge arrow value: " + err.Error())

		a.send("error", "Invalid <strong>edge arrow</strong> value given, boolean expected.", "Can't save!")
		return
	}

	smooth, err := strconv.ParseBool(parts[11])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse edge smooth value: " + err.Error())

		a.send("error", "Invalid <strong>edge smooth</strong> value given, boolean expected.", "Can't save!")
		return
	}

	hover, err := strconv.ParseBool(parts[12])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse hover value: " + err.Error())

		a.send("error", "Invalid <strong>hover</strong> value given, boolean expected.", "Can't save!")
		return
	}

	multiselect, err := strconv.ParseBool(parts[13])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse multiselect value: " + err.Error())

		a.send("error", "Invalid <strong>multiselect</strong> value given, boolean expected.", "Can't save!")
		return
	}

	hideedgesondrag, err := strconv.ParseBool(parts[14])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse hideedgesondrag value: " + err.Error())

		a.send("error", "Invalid <strong>hide edges on drag</strong> value given, boolean expected.", "Can't save!")
		return
	}

	// Update graph settings
	opt := &GraphSettings{
		ID: "graph",

		NodeSize:        nodesize,
		BorderWidth:     borderwidth,
		BGcolor:         bgcolor,
		BorderColor:     bordercolor,
		NodeFontSize:    nodefontsize,
		Shadow:          shadow,
		EdgeWidth:       edgewidth,
		EdgeColor:       edgecolor,
		EdgeFontSize:    edgefontsize,
		EdgeFontColor:   edgefontcolor,
		Arrow:           arrow,
		Smooth:          smooth,
		Hover:           hover,
		MultiSelect:     multiselect,
		HideEdgesOnDrag: hideedgesondrag,
	}

	err = db.setGraphSettings(opt)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't update graph UI settings: " + err.Error())

		a.send("error", "Can't update graph UI settings: "+err.Error(), "Can't save!")
		return
	}

	a.send("ok", "", "")

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Graph UI settings updated")
}

/*
 * Handle 'users' websocket command to manage users.
 *
 * Acceptable commands:
 *   reset-password: drop user's password, so he can renew it
 *   delete:         delete specific user
 *   admin-true:     give admin rights
 *   admin-false:    remove admin rights
 */
func (a *Account) usersHandler(data string) {

	// Drop non-admin users request
	if !a.Admin && config.Environment == "prod" {
		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Not enough rights to set users options")

		a.send("error", "Not enough rights to set users options.", "Error!")
		return
	}

	parts := strings.Split(data, ",")

	// Get action & user to modify
	user := parts[0]
	action := parts[1]

	// Get an account to modify from a database
	userAccount, err := db.getAccount(user)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't GetAccount to get user to modify: " + err.Error())

		a.send("error", "Can't validate requested user: "+err.Error(), "Error!")
		return
	}

	switch action {
	case "reset-password":
		err = userAccount.update("password", "")
		if err != nil {
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Str("user-to-reset", user).
				Msg("Can't reset password: " + err.Error())

			a.send("error", "Can't reset password: "+err.Error(), "Error!")
			return
		}

		a.send("account-reset", "", "")

		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Str("user-to-mod", user).
			Msg("Password was reset")

	case "delete":
		err = db.deleteAccount(user)
		if err != nil {
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Str("user-to-delete", user).
				Msg("Can't deleteAccount: " + err.Error())

			a.send("error", "Can't delete account: "+err.Error(), "Error!")
			return
		}

		// Delete session if exists
		err = db.deleteSession(user, nil, nil)
		if err != nil && err != errSessionNotExists {
			log.Error().
				Str("username", a.Username).
				Str("user-to-delete", user).
				Msg("Can't delete user session: " + err.Error())

			a.send("error", "Can't delete user session: "+err.Error(), "Error!")
			return
		}

		a.send("account-deleted", user, "")

		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Str("user-to-mod", user).
			Msg("User deleted")

	case "admin-true":
		err = userAccount.update("admin", true)
		if err != nil {
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Str("user-to-make-admin", user).
				Msg("Can't set admin rights: " + err.Error())

			a.send("error", "Can't set admin rights: "+err.Error(), "Error!")
			return
		}

		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Str("user-to-mod", user).
			Msg("User got admin rights")

	case "admin-false":
		err = userAccount.update("admin", false)
		if err != nil {
			log.Error().
				Str("ip", a.Session.IP).
				Str("username", a.Username).
				Str("user-to-remove-admin", user).
				Msg("Can't unset admin rights: " + err.Error())

			a.send("error", "Can't unset admin rights: "+err.Error(), "Error!")
			return
		}

		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Str("user-to-mod", user).
			Msg("User lost admin rights")

	default:
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Unknown user mod. action requested: " + action)

		a.send("error", "Unknown user mod. action requested: "+action, "Error!")
		return
	}
}

/*
 * Reload collectors.
 *
 * This allows:
 *   - To recreate dropped connections to the data sources
 *   - To refresh the list of fields to query for the Web GUI autocomplete
 */
func (a *Account) reloadCollectorsHandler() {
	err := setupCollectors()
	if err != nil {
		a.send("error", "Can't reload collectors: "+err.Error(), "Error!")

		log.Info().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't reload collectors: " + err.Error())
		return
	}

	a.send("ok", "", "")

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Collectors reloaded")
}
