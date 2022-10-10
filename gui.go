package main

import (
	"errors"
	"net/http"
)

var (
	// Nodes styling definition.
	// Will be applied to the JavaScript engine
	groups string
)

/*
 * Setup Web GUI handlers
 */
func setupGUI() error {
	var err error

	// Parse graph elements style groups definitions
	groups, err = loadFileIntoString(config.Groups)
	if err != nil {
		return errors.New("Can't load groups file: " + err.Error())
	}

	// Load query formatting rules,
	// which help to format comma/space separated indicators to a valid SQL query
	err = loadFormats()
	if err != nil {
		return errors.New("Can't load query formatting rules: " + err.Error())
	}

	// Create a sessions store
	err = setupSessions()
	if err != nil {
		return errors.New("Can't setup sessions store: " + err.Error())
	}

	// Parse documentation files
	err = loadDocs()
	if err != nil {
		return errors.New("Can't load documentation: " + err.Error())
	}

	// Notify users about new features
	err = loadFeatures()
	if err != nil {
		return errors.New("Can't show new features: " + err.Error())
	}

	// Setup the indicators upload feature
	err = setupUpload()
	if err != nil {
		return errors.New("Can't setup the indicators upload feature: " + err.Error())
	}

	// Setup additional HTTPS handlers
	// which are required by a Web GUI
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/signin", signinHandler)
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/signout", signoutHandler)
	http.HandleFunc("/profile", profileHandler)
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/docs", docsHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/download", downloadHandler)
	http.HandleFunc("/ws", wsHandler)

	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("assets"))))

	return nil
}
