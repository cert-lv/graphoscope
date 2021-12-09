package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	deflog "log"

	"github.com/go-stuff/mongostore"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var (
	// Hold signed in users sessions
	sessions *Sessions

	// Common error to use across the source code
	// when user is not signed in
	errSessionNotExists = fmt.Errorf("No sessions were deleted")
)

/*
 * Structure to hold signed in users sessions
 */
type Sessions struct {
	// Browser's cookie name
	CookieName string

	// Place to store sessions data, MongoDB
	Store *mongostore.Store
}

/*
 * Structure of a single user's session
 */
type Session struct {
	// Client's IP address
	IP string

	// Websocket connection for client-server-client communication
	Websocket *websocket.Conn

	// Client's initial HTTP request
	Request *http.Request

	// Client's initial HTTP response writer
	ResponseWriter http.ResponseWriter

	// A channel to use accross the code
	// to indicate about a closed websocket connection
	Done chan bool
}

/*
 * Create a sessions store when service is started
 */
func setupSessions() error {
	// Disable a default 'mongostore' printing to the console
	deflog.SetOutput(ioutil.Discard)

	// Drop old TTL index
	_, err := db.Sessions.Indexes().DropAll(db.newContext())
	if err != nil {
		// Ignore namespace not found errors
		commandErr, ok := err.(mongo.CommandError)
		if !ok {
			log.Error().Msg("Can't check MongoDB sessions indexes drop error: " + err.Error())
		}
		if commandErr.Name != "NamespaceNotFound" {
			log.Error().Msg("Failed to drop session coll's indexes: " + err.Error())
		}

	} else {
		log.Debug().Msg("Session coll's old indexes are dropped")
	}

	// Create a new store with new index
	store, err := mongostore.NewStore(
		db.Sessions,
		http.Cookie{
			Path:     "/",
			Domain:   "",
			MaxAge:   config.Sessions.TTL,
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		},
		[]byte(config.Sessions.AuthenticationKey),
		[]byte(config.Sessions.EncryptionKey),
	)
	if err != nil {
		return fmt.Errorf("Can't create a MongoDB session store: " + err.Error())
	}

	sessions = &Sessions{
		CookieName: config.Sessions.CookieName,
		Store:      store,
	}

	log.Debug().Msg("MongoDB session store created")
	return nil
}

/*
 * Check whether user is signed in
 */
func (s *Sessions) exists(w http.ResponseWriter, r *http.Request) (string, error) {
	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg("Can't get IP to check existing session: " + err.Error())
		return "", fmt.Errorf("Server internal error, try again later")
	}

	session, err := s.Store.Get(r, s.CookieName)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg("Can't get existing session: " + err.Error())

		return "", fmt.Errorf("Server internal error, try again later")
	}

	//fmt.Printf("session: %#v\n", session)

	if len(session.Values) == 0 {
		// Check a DB connection.
		// 's.Store.Get' doesn't return an error if connection was lost
		err = s.Store.Collection.Database().Client().Ping(db.newContext(), nil)
		if err != nil {
			return "", fmt.Errorf("Can't ping a database: " + err.Error())
		}
	}

	if username, ok := session.Values["username"].(string); ok {
		// Update session
		err = session.Save(r, w)
		if err != nil {
			log.Error().
				Str("ip", ip).
				Msg("Can't update session " + err.Error())

			return "", fmt.Errorf("Server internal error, try again later")
		}

		log.Debug().
			Str("ip", ip).
			Str("username", username).
			Msg("Session updated")

		return username, nil
	}

	return "", fmt.Errorf("Requested by unsigned user: " + r.Host + r.URL.RequestURI())
}

/*
 * Process '/signup' request to register a new account
 */
func signupHandler(w http.ResponseWriter, r *http.Request) {
	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg("Can't get IP to signup: " + err.Error())
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Validate user input
	err = credentialsAreValid(username, password)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg("Invalid credentials: " + err.Error())

		templateData := &TemplateData{
			Error: err.Error(),
		}

		renderTemplate(w, "signin", templateData, nil)
		return
	}

	account := &Account{}
	err = db.Users.FindOne(db.newContext(), bson.M{"username": username}).Decode(account)

	if err != nil {
		/*
		 * Create a new account
		 */
		if err == mongo.ErrNoDocuments {
			err = newAccount(w, r)
			if err != nil {
				log.Error().
					Str("ip", ip).
					Str("username", username).
					Msg("Can't create new account: " + err.Error())

				templateData := &TemplateData{
					Error: "Server internal error, try again later!",
				}

				renderTemplate(w, "signin", templateData, nil)
			} else {
				log.Info().
					Str("ip", ip).
					Str("username", username).
					Msg("Registration successful")

				http.Redirect(w, r, "/", 302)
			}

		} else {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't find username in DB: " + err.Error())

			templateData := &TemplateData{
				Error: "Server internal error, try again later!",
			}

			renderTemplate(w, "signin", templateData, nil)
		}

		/*
		 * Set by admin previously removed password.
		 * Allows to reset forgotten password
		 */
	} else if account.Password == "" {

		err = account.setPassword(r.FormValue("password"))
		if err != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't setPassword: " + err.Error())

			templateData := &TemplateData{
				Error: "Server internal error, try again later!",
			}

			renderTemplate(w, "signin", templateData, nil)
			return
		}

		session, err := sessions.Store.Get(r, sessions.CookieName)
		if err != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't get session to set new password: " + err.Error())

			templateData := &TemplateData{
				Error: "Server internal error, try again later!",
			}

			renderTemplate(w, "signin", templateData, nil)
			return
		}

		// Add username to the session.
		// Without this user can't sign up immediately after password reset
		session.Values["username"] = username

		// Save session
		err = session.Save(r, w)
		if err != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't save session with new password: " + err.Error())

			templateData := &TemplateData{
				Error: "Server internal error, try again later!",
			}

			renderTemplate(w, "signin", templateData, nil)
			return
		}

		log.Info().
			Str("ip", ip).
			Str("username", username).
			Msg("Password updated")

		http.Redirect(w, r, "/", 302)

		/*
		 * Account already exists and is complete
		 */
	} else {
		log.Error().
			Str("ip", ip).
			Msg("Username already taken: " + username)

		templateData := &TemplateData{
			Error: "Username already taken!",
		}

		renderTemplate(w, "signin", templateData, nil)
	}
}

/*
 * Process '/signin' request for the existing account
 */
func signinHandler(w http.ResponseWriter, r *http.Request) {
	// Accept GET requests only
	if r.Method == "GET" {
		templateData := &TemplateData{}
		renderTemplate(w, "signin", templateData, nil)
		return
	}

	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg("Can't get IP to signin: " + err.Error())
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Skip trying empty data
	if username == "" || password == "" {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Input fields can't be empty")

		templateData := &TemplateData{
			Error: "Input fields can't be empty",
		}

		renderTemplate(w, "signin", templateData, nil)
		return
	}

	account := &Account{}

	err = db.Users.FindOne(db.newContext(), bson.M{"username": username}).Decode(account)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msgf("Invalid username '%s': %s", username, err.Error())

		templateData := &TemplateData{
			Error: "Invalid username",
		}

		renderTemplate(w, "signin", templateData, nil)
		return
	}

	// Validate user's password
	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password))
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Invalid password: " + err.Error())

		templateData := &TemplateData{
			Error: "Invalid password",
		}

		renderTemplate(w, "signin", templateData, nil)
		return
	}

	// User is authenticated, create a new session
	session, err := sessions.Store.Get(r, sessions.CookieName)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't get session: " + err.Error())

		templateData := &TemplateData{
			Error: "Server internal error, try again later!",
		}

		renderTemplate(w, "signin", templateData, nil)
		return
	}

	// Add values to the session
	session.Values["username"] = username

	// Save session
	err = session.Save(r, w)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't save session: " + err.Error())

		templateData := &TemplateData{
			Error: "Server internal error, try again later!",
		}

		renderTemplate(w, "signin", templateData, nil)
		return
	}

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Msg("User signed in")

	http.Redirect(w, r, "/", 302)
}

/*
 * Sign out existing user
 */
func signoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg("Can't get IP to signout: " + err.Error())

		http.Redirect(w, r, "/signin", 302)
		return
	}

	// Check existing session
	username, err := sessions.exists(w, r)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Msg("Can't validate session to sign out: " + err.Error())

		fmt.Fprint(w, "Can't validate user session: "+err.Error())
		return
	}

	// Delete session
	err = db.deleteSession(username, w, r)
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't close user session: " + err.Error())

		http.Redirect(w, r, "/signin", 302)
		return
	}

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Msg("User signed out")

	http.Redirect(w, r, "/signin", 302)
}
