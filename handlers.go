package main

import (
	"html/template"
	"net"
	"net/http"

	"github.com/cert-lv/graphoscope/pdk"
)

/*
 * Structure to be inserted in the HTML templates
 * as a dynamic content
 */
type TemplateData struct {
	// A list of connected data sources,
	// will be used to generate sources dropdowns
	Collectors map[string]pdk.SourcePlugin

	// A list of shared dashboards to be loaded
	Shared map[string]*Dashboard

	// Helper variable to hide some HTML elements
	// when non-global data sources don't exist
	NonGlobalExist bool

	// Nodes styling definition.
	// Will be applied to the JavaScript engine
	Groups string

	// Query formatting rules,
	// which help to format comma/space separated indicators to a valid SQL query
	Formats string

	// A list of all known data sources fields for the Web GUI autocomplete
	Fields map[string][]string

	// Built-in documentation content, N sections
	Docs [3]string

	// Currently signed in user
	Account *Account

	// A list of all registered users
	Accounts []*Account

	// A list of new features for the current service's version.
	// Will be displayed once for each user
	Features []string

	// Graph UI settings
	GraphSettings *GraphSettings

	// Service runs in a Development or Production environment
	Environment string

	// The latest service's version
	Version string

	// Possible error to show to the user
	Error string
}

/*
 * Render requested HTML template.
 *
 * Receives a template's name, a dynamic data to insert and
 * functions to run in Go templates
 */
func renderTemplate(w http.ResponseWriter, tmpl string, data *TemplateData, funcs template.FuncMap) {

	// Fill static info
	data.Environment = config.Environment
	data.Version = version

	templates := template.New("").Funcs(funcs)

	// Parse needed HTML template
	templates, err := templates.ParseFiles(
		"assets/tmpl/"+tmpl+".html",
		"assets/tmpl/modal.html",
		"assets/tmpl/topbar.html",
		"assets/tmpl/credits.html",
	)
	if err != nil {
		log.Error().Msg(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		log.Error().Msg(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

/*
 * Serve main '/' web page
 */
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI != "/" {
		return
	}

	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().Msg("Can't get IP for index page: " + err.Error())
		return
	}

	// Check existing session
	username, err := sessions.exists(w, r)
	if err != nil {
		log.Debug().
			Str("ip", ip).
			Msg(err.Error())

		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Get account from a database
	account, err := db.getAccount(username)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't get account to get filters: " + err.Error())

		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// All are admins in a development environment
	if config.Environment != "prod" {
		account.Admin = true
	}

	// Get account from a database
	shared, err := db.getSharedDashboards()
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't get shared dashboards: " + err.Error())
	}

	// Get Web UI settings
	settings, err := db.getGraphSettings()
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't get graph UI settings: " + err.Error())

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Collect dynamic data
	templateData := &TemplateData{
		Account:        account,
		Collectors:     collectors,
		NonGlobalExist: nonGlobalExist,
		Shared:         shared,
		Groups:         groups,
		Formats:        formats,
		Fields:         fields,
		GraphSettings:  settings,
	}

	if account.SeenFeatures != features[0] {
		templateData.Features = features
	}

	renderTemplate(w, "index", templateData, nil)
	account.hideFeatures()

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Msg("Index page requested")
}
