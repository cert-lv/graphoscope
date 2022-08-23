package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

/*
 * Structure to store all the info about a registered user/account
 */
type Account struct {
	// Username to sign in
	Username string `bson:"username"`

	// Hash of the password
	Password string `bson:"password"`

	// Unique auth ID to use in API requests
	UUID string `bson:"uuid"`

	// Whether user has admin rights
	Admin bool `bson:"admin"`

	// A list of current filters.
	// Not 'string' to prevent the value to be quoted in a HTML template.
	// Is a map of "SQL query" -> filter's state
	Filters []map[string]*Filter `bson:"filters"`

	// A list of saved dashboards which can be loaded.
	// Is a map of "dashboard name" -> its content
	Dashboards map[string]*Dashboard `bson:"dashboards"`

	// Date when new features were seen the last time,
	// to see the notification only once
	SeenFeatures string `bson:"seenFeatures"`

	// Personal settings
	// which override the default server-side settings
	Options *Options `bson:"options"`

	// A list of uploaded and processed files with indicators
	Uploads *Uploads `bson:"uploads"`

	// Whether to show a debug info of the queries
	Debug bool `bson:"debug"`

	// Personal notifications if user wasn't online and
	// wasn't able to receive a notification in a real time
	Notifications []*Notification `bson:"notifications"`

	// Timestamp when user was active the last time
	LastActive time.Time `bson:"lastActive"`

	// Current session's info,
	// will not be stored in a database
	Session *Session `bson:"-"`
}

/*
 * A list of uploaded and processed files with indicators
 */
type Uploads struct {
	// Queue of the files to be processed
	In map[string]*Upload `bson:"in"`

	// A list of processed files.
	// Will be cleaned periodically
	Out []string `bson:"out"`

	// Max size of the uploaded file in bytes
	MaxSize int64 `bson:"-"`
}

/*
 * Personal settings
 * which override the default server-side settings.
 * Configurable through the Web GUI settings -> "Profile"
 */
type Options struct {
	// A way to enable/disable initial graph animations when new nodes are added
	StabilizationTime int `bson:"stabilizationTime"`

	// The amount of entries each data source should return.
	// Will be a part of each SQL query, set to 0 to disable
	Limit int `bson:"limit"`

	// Whether to display queries debug info
	Debug bool `bson:"debug"`
}

/*
 * Create a new account,
 * store it in a database and sign up the user
 */
func newAccount(w http.ResponseWriter, r *http.Request) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), 5)
	if err != nil {
		return fmt.Errorf("Error while hashing password: " + err.Error())
	}

	account := &Account{
		Username:     r.FormValue("username"),
		Password:     string(hash),
		UUID:         uuid.NewString(),
		Filters:      []map[string]*Filter{make(map[string]*Filter), make(map[string]*Filter)},
		Dashboards:   make(map[string]*Dashboard),
		SeenFeatures: features[0], // Do not show new features after signing up
		Options: &Options{
			StabilizationTime: config.StabilizationTime,
			Limit:             0,
		},
		Uploads: &Uploads{
			In:  make(map[string]*Upload),
			Out: []string{},
		},
		Notifications: []*Notification{},
		LastActive:    time.Now(),
	}

	_, err = db.Users.InsertOne(db.newContext(), account)
	if err != nil {
		return fmt.Errorf("Error while inserting user: " + err.Error())
	}

	session, err := sessions.Get(r, config.Sessions.CookieName)
	if err != nil {
		return fmt.Errorf("Can't get session: " + err.Error())
	}

	// Add username to the session
	session.Values["username"] = r.FormValue("username")

	// Save session
	err = session.Save(r, w)
	if err != nil {
		return fmt.Errorf("Can't save session: " + err.Error())
	}

	return nil
}

/*
 * Set a new password
 */
func (a *Account) setPassword(password string) error {
	// Validate user input
	err := credentialsAreValid(a.Username, password)
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	// Get hash from the password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 5)
	if err != nil {
		return fmt.Errorf("Error while hashing new password: " + err.Error())
	}

	err = a.update("password", string(hash))
	if err != nil {
		return fmt.Errorf("Can't update account to set new password: " + err.Error())
	}

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Password updated")

	return nil
}

/*
 * Checks whether credentials are strong enough
 */
func credentialsAreValid(username, password string) error {
	if len(username) < 5 || len(username) > 20 {
		return fmt.Errorf("Username must be 5-20 characters")
	}

	if len(password) < 8 {
		return fmt.Errorf("Password must be at least 8 characters long")
	}

	return nil
}

/*
 * Update struct fields in case its structure has changed.
 * For backward compatibility.
 */
func (a *Account) adoptFields() error {
	if a.Options == nil {
		a.Options = &Options{
			StabilizationTime: config.StabilizationTime,
			Limit:             config.Limit,
		}

		err := a.update("options", a.Options)
		if err != nil {
			return fmt.Errorf("Can't update account to set 'a.Options': " + err.Error())
		}
	}

	if a.Uploads == nil {
		a.Uploads = &Uploads{
			In:  make(map[string]*Upload),
			Out: []string{},
		}

		err := a.update("uploads", a.Uploads)
		if err != nil {
			return fmt.Errorf("Can't update account to set 'a.Uploads': " + err.Error())
		}
	}

	if a.Notifications == nil {
		a.Notifications = []*Notification{}

		err := a.update("notifications", a.Notifications)
		if err != nil {
			return fmt.Errorf("Can't update account to set 'a.Notifications': " + err.Error())
		}
	}

	return nil
}

/*
 * Handle 'uuid' websocket command to regenerate a new auth UUID
 */
func (a *Account) regenerateUUID() {
	a.UUID = uuid.NewString()

	err := a.update("uuid", a.UUID)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't regenerate UUID: " + err.Error())

		a.send("error", err.Error(), "Can't regenerate UUID!")
		return
	}

	a.send("uuid", a.UUID, "")

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("UUID regenerated")
}

/*
 * Handle 'account-save' websocket command to save account data
 */
func (a *Account) saveHandler(password string) {
	// Update password
	err := a.setPassword(password)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't update password: " + err.Error())

		a.send("error", err.Error(), "Can't update password!")
		return
	}

	a.send("ok", "", "")

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Account data updated")
}

/*
 * Handle 'account-delete' websocket command to delete an account
 */
func (a *Account) delete() {
	// Delete account from a database
	err := db.deleteAccount(a.Username)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't delete account: " + err.Error())

		a.send("error", err.Error(), "Can't delete account!")
		return
	}

	// Delete session
	err = db.deleteSession(a.Username, a.Session.ResponseWriter, a.Session.Request)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't delete user session: " + err.Error())

		a.send("error", err.Error(), "Can't delete session!")
		return
	}

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Account deleted")
}

/*
 * Update one account's field in a database
 */
func (a *Account) update(field string, value interface{}) error {
	filter := bson.M{"username": a.Username}
	update := bson.M{"$set": bson.M{
		field: value,
	}}

	result, err := db.Users.UpdateOne(db.newContext(), filter, update)
	if result.ModifiedCount == 0 {
		log.Error().Msgf("'%s' was not updated for any account", field)
		return err
	}
	if err != nil {
		log.Error().Msg("Can't update account field: " + err.Error())
		return err
	}

	// Additional check in case an action was made by the admin
	if a.Session != nil {
		log.Debug().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msgf("Account's '%s' is updated in a database", field)
	} else {
		log.Debug().
			Str("username", a.Username).
			Msgf("Account's '%s' is updated in a database", field)
	}

	return nil
}
