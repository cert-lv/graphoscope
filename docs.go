package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
)

var (
	// Built-in documentation content, N sections
	docs [3]string
)

/*
 * Serve '/docs' page with a built-in documentation
 */
func docsHandler(w http.ResponseWriter, r *http.Request) {
	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().Msg("Can't get IP for the docs page: " + err.Error())
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
			Msg("Can't GetAccount for the docs page: " + err.Error())

		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// All are admins in service's DEV mode
	if config.Environment != "prod" {
		account.Admin = true
	}

	templateData := &TemplateData{
		Account: account,
		Docs:    docs,
	}

	renderTemplate(w, "docs", templateData, nil)

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Msg("Docs page requested")
}

/*
 * Load all documentation from the markdown files.
 * One file for one documentation section
 */
func loadDocs() error {
	for i, name := range []string{"ui", "search", "admin"} {
		md, err := loadDoc(name)
		if err != nil {
			return err
		}

		docs[i] = md
	}

	log.Debug().Msg("Documentation is parsed")
	return nil
}

/*
 * Load documentation from a specific markdown file.
 * Receives a name of file to load, its extension has to be ".md"
 */
func loadDoc(name string) (string, error) {
	mdFile, err := os.Open(config.Docs + "/" + name + ".md")
	if err != nil {
		return "", fmt.Errorf("Failed to open '%s/%s.md' doc: %s", config.Docs, name, err.Error())
	}

	fi, _ := mdFile.Stat()
	buffer := make([]byte, fi.Size())
	_, err = mdFile.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("Failed to read '%s/%s.md' doc: %s", config.Docs, name, err.Error())
	}

	return string(buffer), nil
}
