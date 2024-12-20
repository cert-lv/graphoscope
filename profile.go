package main

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

/*
 * Serve '/profile' page
 * with profile settings and long term actions
 */
func profileHandler(w http.ResponseWriter, r *http.Request) {
	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().Msg("Can't get IP for profile page: " + err.Error())
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

	// Get account from a database
	account, err := db.getAccount(username)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't GetAccount for profile page: " + err.Error())

		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// All are admins in service's development mode
	if config.Environment != "prod" {
		account.Admin = true
	}

	// Merge some settings
	account.Uploads.MaxSize = config.Upload.MaxSize

	templateData := &TemplateData{
		Account:        account,
		Collectors:     collectors,
		NonGlobalExist: nonGlobalExist,
	}

	renderTemplate(w, "profile", templateData, nil)

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Msg("Profile page requested")
}

/*
 * Handle 'options' websocket command to save personal settings.
 *
 * Receives a comma-separated values:
 *   1. New nodes stabilization time in milliseconds
 *   2. Limit value as a part of each query to the data sources
 */
func (a *Account) optionsHandler(data string) {

	parts := strings.Split(data, ",")

	stabilization, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse stabilization time value: " + err.Error())

		a.send("error", "Invalid <strong>stabilization time</strong> value given, integer expected.", "Can't save!")
		return
	}

	limit, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse limit value: " + err.Error())

		a.send("error", "Invalid <strong>limit</strong> value given, integer expected.", "Can't save!")
		return
	}

	showLimited, err := strconv.ParseBool(parts[2])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse showLimited value: " + err.Error())

		a.send("error", "Invalid <strong>showLimited</strong> value given, boolean expected.", "Can't save!")
		return
	}

	debug, err := strconv.ParseBool(parts[3])
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't parse debug value: " + err.Error())

		a.send("error", "Invalid <strong>debug</strong> value given, boolean expected.", "Can't save!")
		return
	}

	// Update options
	options := &Options{
		StabilizationTime: stabilization,
		Limit:             limit,
		ShowLimited:       showLimited,
		Debug:             debug,
	}

	err = a.update("options", options)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't update profile options: " + err.Error())

		a.send("error", err.Error(), "Can't update profile options!")
		return
	}

	a.send("ok", "", "")

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Profile options updated")
}
