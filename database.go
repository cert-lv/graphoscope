package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	// Service's local database
	db *Database

	// Last digit of minutes and seconds will be removed
	// from the queries as keys to make caching to work.
	// Otherwise every single refresh will produce a new cache entry
	reDatetimeLimit = regexp.MustCompile(`( AND datetime BETWEEN '\d{4}-\d{2}-\d{2}T\d{2}:\d)[\d:]{4}(\.000Z' AND '\d{4}-\d{2}-\d{2}T\d{2}:\d)[\d:]{4}(\.000Z')( LIMIT (\d*,)?\d*)?$`)
)

/*
 * Structure to hold access to the collections of the local database
 */
type Database struct {
	// When user signs in a new session is created and added to
	// this collection. Sessions expire after a predefined time
	Sessions *mongo.Collection

	// Registered users
	Users *mongo.Collection

	// Saved shared dasboards
	Dashboards *mongo.Collection

	// Users notes for the graph elements
	Notes *mongo.Collection

	// Cached requests and results for a faster response
	// when identical request happens
	Cache *mongo.Collection

	// Graph global UI settings
	Settings *mongo.Collection
}

/*
 * Structure to describe a cache entry
 */
type Cache struct {
	// Graph relations data
	Relations []map[string]interface{} `bson:"relations"`

	// Statistics info
	Stats map[string]interface{} `bson:"stats"`

	// Record creation timestamp for the TTL
	Ts time.Time `bson:"ts"`
}

/*
 * Create a connection to the database and its collections
 */
func setupDatabase() error {
	// Database log in credentials
	credential := options.Credential{
		AuthSource: config.Database.Name,
		Username:   config.Database.User,
		Password:   config.Database.Password,
	}

	client, err := mongo.NewClient(options.Client().
		SetAuth(credential).
		ApplyURI(config.Database.URL))
	if err != nil {
		return fmt.Errorf("Can't create a MongoDB client: " + err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Database.Timeout)*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return fmt.Errorf("Can't connect to the database: " + err.Error())
	}

	// Check the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("Can't ping the database: " + err.Error())
	}

	// Set global variable
	db = &Database{
		Sessions:   client.Database(config.Database.Name).Collection(config.Database.Sessions),
		Users:      client.Database(config.Database.Name).Collection(config.Database.Users),
		Dashboards: client.Database(config.Database.Name).Collection(config.Database.Dashboards),
		Notes:      client.Database(config.Database.Name).Collection(config.Database.Notes),
		Cache:      client.Database(config.Database.Name).Collection(config.Database.Cache),
		Settings:   client.Database(config.Database.Name).Collection(config.Database.Settings),
	}

	db.prepare()
	db.setCacheTTL()

	log.Debug().Msg("Database successfully connected")

	return nil
}

/*
 * Prepare initial database content
 * in case of a fresh installation or a new collection
 */
func (d *Database) prepare() {
	// Setup graph UI settings
	settings := &GraphSettings{}
	filter := bson.M{"_id": "graph"}

	err := d.Settings.FindOne(d.newContext(), filter).Decode(settings)
	if err == mongo.ErrNoDocuments {
		settings = &GraphSettings{
			ID: "graph",

			NodeSize:        10,
			BorderWidth:     1,
			BGcolor:         "#f00",
			BorderColor:     "#000",
			NodeFontSize:    20,
			Shadow:          true,
			EdgeWidth:       2,
			EdgeColor:       "#f00",
			EdgeFontSize:    16,
			EdgeFontColor:   "#888",
			Arrow:           true,
			Smooth:          false,
			Hover:           true,
			MultiSelect:     true,
			HideEdgesOnDrag: false,
		}

		_, err := d.Settings.InsertOne(d.newContext(), settings)
		if err != nil {
			log.Error().Msg("Can't prepare graph UI settings: " + err.Error())
			return
		}

		log.Info().Msg("Graph UI settings prepared")
	}
}

/*
 * Registered users management
 */

/*
 * Return account by its name
 */
func (d *Database) getAccount(name string) (*Account, error) {
	account := &Account{}
	filter := bson.M{"username": name}

	err := d.Users.FindOne(d.newContext(), filter).Decode(account)
	if err != nil {
		return nil, err
	}

	// Update struct fields in case database entry's fields has changed
	err = account.adoptFields()
	if err != nil {
		return nil, err
	}

	// Update user's last active time
	err = account.update("lastActive", time.Now())
	if err != nil {
		return nil, fmt.Errorf("Can't update account to set 'lastActive' time: " + err.Error())
	}

	return account, nil
}

/*
 * Return account by its UUID
 */
func (d *Database) getAccountByUUID(uuid string) (*Account, error) {
	account := &Account{}
	filter := bson.M{"uuid": uuid}

	err := d.Users.FindOne(d.newContext(), filter).Decode(account)
	if err != nil {
		return nil, err
	}

	// Update struct fields in case database entry's fields has changed
	err = account.adoptFields()
	if err != nil {
		return nil, err
	}

	// Update user's last active time
	err = account.update("lastActive", time.Now())
	if err != nil {
		return nil, fmt.Errorf("Can't update account to set 'lastActive' time: " + err.Error())
	}

	return account, nil
}

/*
 * Return all accounts
 */
func (d *Database) getAccounts() ([]*Account, error) {
	accounts := []*Account{}

	// Find all entries
	cursor, err := d.Users.Find(d.newContext(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(d.newContext())

	// Decode results one by one
	for cursor.Next(d.newContext()) {
		account := &Account{}

		err := cursor.Decode(&account)
		if err != nil {
			return nil, err
		}

		// Update struct fields in case database entry's fields has changed
		err = account.adoptFields()
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	// Sort by usernames.
	// MongoDB itselt sorting is numeric first, then upper case letters, then lower case letters last,
	// but we want it to be case insensitive
	sort.Slice(accounts, func(i, j int) bool {
		return strings.ToLower(accounts[i].Username) < strings.ToLower(accounts[j].Username)
	})

	// Return accounts
	return accounts, nil
}

/*
 * Delete account by its username
 */
func (d *Database) deleteAccount(username string) error {
	filter := bson.M{"username": username}

	res, err := d.Users.DeleteOne(d.newContext(), filter)

	if res.DeletedCount == 0 {
		log.Error().
			Str("username", username).
			Msg("No accounts were deleted")

		return fmt.Errorf("No accounts were deleted")
	}

	return err
}

/*
 * Delete user session by its username
 * when user signs out or is deleted.
 */
func (d *Database) deleteSession(username string, w http.ResponseWriter, r *http.Request) error {
	// Delete from a database
	filter := bson.M{"data": bson.M{"username": username}}

	res, err := d.Sessions.DeleteOne(d.newContext(), filter)
	if res.DeletedCount == 0 {
		log.Debug().
			Str("username", username).
			Msg("No sessions were deleted")

		return errSessionNotExists
	}
	if err != nil {
		return fmt.Errorf("Can't delete session: " + err.Error())
	}

	// Delete Web session if exists
	if r != nil {
		session, err := sessions.Get(r, config.Sessions.CookieName)
		if err != nil {
			return fmt.Errorf("Can't get session to delete: " + err.Error())
		}

		session.Options.MaxAge = -1

		if err = session.Save(r, w); err != nil {
			return err
		}
	}

	// Close Websocket connection if exists
	if account, exists := online[username]; exists {
		account.Session.Websocket.Close()
	}

	return nil
}

/*
 * Return all shared dashboards.
 * Return at least empty map to be able to iterate it
 */
func (d *Database) getSharedDashboards() (map[string]*Dashboard, error) {
	dashboards := make(map[string]*Dashboard)

	// Find all entries
	cursor, err := d.Dashboards.Find(d.newContext(), bson.M{})
	if err != nil {
		return dashboards, err
	}
	defer cursor.Close(d.newContext())

	// Decode results one by one
	for cursor.Next(d.newContext()) {
		result := &Dashboard{}

		err := cursor.Decode(&result)
		if err != nil {
			return dashboards, err
		}

		dashboards[result.Name] = result
	}
	if err := cursor.Err(); err != nil {
		return dashboards, err
	}

	return dashboards, nil
}

/*
 * Manage users notes for the graph elements
 */

/*
 * Get notes for the given graph element by its value
 */
func (d *Database) getNotes(id string) (string, error) {
	note := make(map[string]string)
	filter := bson.M{"_id": id}

	err := d.Notes.FindOne(d.newContext(), filter).Decode(&note)
	if err == mongo.ErrNoDocuments {
		log.Debug().
			Str("id", id).
			Msg("Element notes do not exist in a database")
		return "", nil

	} else if err != nil {
		return "", err
	}

	log.Debug().
		Str("id", id).
		Str("notes", note["value"]).
		Msg("Notes found")

	return note["value"], nil
}

/*
 * Set notes for the given graph element.
 * Receives graph element's value and a note to apply
 */
func (d *Database) setNotes(id, value string) error {
	// Delete note if new value is empty
	if value == "" {
		err := d.delNotes(id)
		if err != nil {
			return err
		}

	} else {
		note := &bson.M{
			"_id":   id,
			"value": value,
		}

		filter := bson.M{"_id": id}
		update := bson.M{"$set": note}

		upsert := true
		opts := &options.UpdateOptions{
			Upsert: &upsert,
		}

		_, err := d.Notes.UpdateOne(d.newContext(), filter, update, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
 * Delete notes for the given graph element by its value
 */
func (d *Database) delNotes(id string) error {
	filter := bson.M{"_id": id}

	res, err := d.Notes.DeleteOne(d.newContext(), filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		log.Debug().
			Str("id", id).
			Msg("No notes were deleted")

		return fmt.Errorf("No notes were deleted")
	}

	return nil
}

/*
 * Manage cache
 */

/*
 * Get cache value by a query text
 */
func (d *Database) getCache(query string) (*Cache, error) {
	cache := &Cache{}
	filter := bson.M{"_id": reDatetimeLimit.ReplaceAllString(query, "$1.:..$2.:..$3$4")}

	err := d.Cache.FindOne(d.newContext(), filter).Decode(cache)
	if err == mongo.ErrNoDocuments {
		log.Debug().
			Str("query", query).
			Msg("Key does not exist in cache")
		return nil, nil

	} else if err != nil {
		return nil, err
	}

	return cache, nil
}

/*
 * Cache the data sources responses.
 * Receives user's query as a key, relations and statistics from data sources
 */
func (d *Database) setCache(query string, relations []map[string]interface{}, stats map[string]interface{}) {
	cache := &Cache{
		Relations: relations,
		Stats:     stats,
		Ts:        time.Now(),
	}

	filter := bson.M{"_id": reDatetimeLimit.ReplaceAllString(query, "$1.:..$2.:..$3$4")}
	update := bson.M{"$set": cache}

	upsert := true
	opts := &options.UpdateOptions{
		Upsert: &upsert,
	}

	// Sometimes identical operations happen concurrently and
	// the same MongoDB key may appear again, so use "UpdateOne" instead of "InsertOne"
	_, err := d.Cache.UpdateOne(d.newContext(), filter, update, opts)
	if err != nil {
		log.Error().Msgf("Can't save '%s' relations data in cache: %s", query, err.Error())
	} else {
		log.Info().
			Str("query", query).
			Msg("Cache set")
	}
}

/*
 * Set TTL for the cache collection's entries
 */
func (d *Database) setCacheTTL() {
	// Drop old index first.
	// Otherwise TTL param won't be updated
	_, err := d.Cache.Indexes().DropAll(d.newContext())
	if err != nil {
		// Ignore namespace not found errors
		commandErr, ok := err.(mongo.CommandError)
		if !ok {
			log.Error().Msg("Can't check MongoDB cache indexes drop error: " + err.Error())
		}
		if commandErr.Name != "NamespaceNotFound" {
			log.Error().Msg("Failed to drop cache coll's indexes: " + err.Error())
		}

	} else {
		log.Debug().Msg("Cache coll's old indexes are dropped")
	}

	// Create a new index
	opts := options.CreateIndexes().SetMaxTime(time.Duration(config.Database.Timeout) * time.Second)

	index := mongo.IndexModel{
		Keys: bson.M{
			"ts": 1,
		},
		Options: &options.IndexOptions{
			ExpireAfterSeconds: &config.Database.CacheTTL,
		},
	}

	_, err = d.Cache.Indexes().CreateOne(d.newContext(), index, opts)
	if err != nil {
		log.Error().Msg("Can't create cache coll's index: " + err.Error())
	} else {
		log.Debug().Msg("Cache coll's index is created")
	}
}

/*
 * UI settings
 */

/*
 * Update graph UI settings.
 * Receives an object with all possible settings
 */
func (d *Database) setGraphSettings(opt *GraphSettings) error {
	filter := bson.M{"_id": "graph"}
	update := bson.M{"$set": opt}

	upsert := true
	opts := &options.UpdateOptions{
		Upsert: &upsert,
	}

	_, err := d.Settings.UpdateOne(d.newContext(), filter, update, opts)
	if err != nil {
		log.Error().Msg("Can't update graph UI settings: " + err.Error())
		return err
	}

	return nil
}

/*
 * Get graph UI settings
 */
func (d *Database) getGraphSettings() (*GraphSettings, error) {
	settings := &GraphSettings{}
	filter := bson.M{"_id": "graph"}

	err := d.Settings.FindOne(d.newContext(), filter).Decode(settings)

	return settings, err
}

/*
 * Create a new context with expiration.
 * Should be used for all database operations
 */
func (d *Database) newContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(config.Database.Timeout)*time.Second)
	return ctx
}
