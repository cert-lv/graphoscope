/*
 * Fork of the original Mongostore at:
 * https://github.com/go-stuff/mongostore/blob/master/mongostore.go
 *
 * Changes of the current version:
 *     - create a new session even if can't decode existing one
 *     - don't print debugging info to the stderr, use logger instead
 *
 * Don't use "log.SetOutput(ioutil.Discard)" to hide printing
 * to the stderr, because it also hides panic errors in other places
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	gsessions "github.com/gorilla/sessions"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/x/bsonx"
)

// MongoSession is how sessions are stored in MongoDB.
type MongoSession struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Data     primitive.M        `bson:"data,omitempty"`
	Modified primitive.DateTime `bson:"modified_at,omitempty"`
	Expires  primitive.DateTime `bson:"expires_at,omitempty"`
	TTL      primitive.DateTime `bson:"ttl,omitemtpy"`
}

// Store stores sessions in Secure Cookies and MongoDB.
type MongoStore struct {
	defaultCookie http.Cookie // default cookie settings
	gsessions.CookieStore

	Context    context.Context
	Collection *mongo.Collection
}

// NewStore uses cookies and mongo to store sessions.
//
// Keys are defined in pairs to allow key rotation, but the common case is
// to set a single authentication key and optionally an encryption key.
//
// The first key in a pair is used for authentication and the second for
// encryption. The encryption key can be set to nil or omitted in the last
// pair, but the authentication key is required in all pairs.
//
// It is recommended to use an authentication key with 32 or 64 bytes.
// The encryption key, if set, must be either 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256 modes.
func NewMongoStore(col *mongo.Collection, cookie http.Cookie, keyPairs ...[]byte) (*MongoStore, error) {
	s := &MongoStore{
		defaultCookie: cookie,
		CookieStore: gsessions.CookieStore{
			Codecs: securecookie.CodecsFromPairs(keyPairs...),
			Options: &gsessions.Options{
				Path:     cookie.Path,
				Domain:   cookie.Domain,
				MaxAge:   cookie.MaxAge,
				Secure:   cookie.Secure,
				HttpOnly: cookie.HttpOnly,
				SameSite: cookie.SameSite,
			},
		},

		Context:    context.Background(),
		Collection: col,
	}

	// add TTL index if it does not exist
	err := s.insertTTL()
	if err != nil {
		return nil, fmt.Errorf("MongoStore: can't add time to live index: %v", err)
	}

	return s, nil
}

// Get returns a session for the given name after adding it to the registry.
//
// It returns a new session if the sessions doesn't exist. Access IsNew on
// the session to check if it is an existing session or a new one.
//
// It returns a new session and an error if the session exists but could
// not be decoded.
func (s *MongoStore) Get(r *http.Request, name string) (*gsessions.Session, error) {
	return gsessions.GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
//
// The difference between New() and Get() is that calling New() twice will
// decode the session data twice, while Get() registers and reuses the same
// decoded session after the first call.
func (s *MongoStore) New(r *http.Request, name string) (*gsessions.Session, error) {
	session := gsessions.NewSession(s, name)
	session.Options = s.CookieStore.Options
	session.Options.MaxAge = s.defaultCookie.MaxAge
	session.IsNew = true

	// get session cookie
	c, err := r.Cookie(name)

	// no cookie
	if errors.Is(err, http.ErrNoCookie) {
		log.Debug().Msgf("MongoStore: no cookie: %s", err.Error())
		return session, nil
	}

	// decode the session.ID in the cookie and use it to find the existing session in mongo
	err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.CookieStore.Codecs...)
	if err != nil {
		return session, fmt.Errorf("MongoStore: can't decode cookie: %w", err)
	}

	// if the session does not exist in mongo, expire the cookies and mark the session as new
	err = s.findOne(session)
	if errors.Is(err, mongo.ErrNoDocuments) {
		log.Debug().Msgf("MongoStore: no session: %s", err.Error())
		return session, nil
	}

	// flag as an existing session
	session.IsNew = false

	return session, nil
}

// Save adds a single session to the response.
func (s *MongoStore) Save(r *http.Request, w http.ResponseWriter, session *gsessions.Session) error {
	// expired session
	if session.Options.MaxAge == -1 {
		res, err := s.deleteOne(session)
		if err != nil {
			return fmt.Errorf("MongoStore: can't delete session: %v", err)
		}
		log.Debug().Msgf("MongoStore: %d session(s) deleted", res.DeletedCount)

	}

	// new session
	if session.IsNew && session.Options.MaxAge != -1 {
		res, err := s.insertOne(session)
		if err != nil {
			return fmt.Errorf("MongoStore: can't insert session: %v", err)
		}
		log.Debug().Msgf("MongoStore: session id: %s, inserted", res.InsertedID.(primitive.ObjectID).Hex())
		session.ID = res.InsertedID.(primitive.ObjectID).Hex()
	}

	// existing session
	if !session.IsNew && session.Options.MaxAge != -1 {
		res, err := s.updateOne(session)
		if err != nil {
			return fmt.Errorf("MongoStore: can't update session: %v", err)
		}
		log.Debug().Msgf("MongoStore: %d session(s) updated", res.ModifiedCount)
	}

	// encode the cookie with only the session.ID, session.Values are never encoded with
	// to the cookie (client side) they are only stored in mongo (server side)
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.CookieStore.Codecs...)
	if err != nil {
		return fmt.Errorf("MongoStore: can't save cookie: %v", err)
	}

	// update the cookie
	http.SetCookie(w, gsessions.NewCookie(session.Name(), encoded, s.CookieStore.Options))

	return nil
}

func (s *MongoStore) insertTTL() error {
	var foundTTLIndex bool

	// get indexes from mongo into the cursor
	cursor, err := s.Collection.Indexes().List(s.Context)
	if err != nil {
		return err
	}

	// use the cursor to iterate each index
	for cursor.Next(s.Context) {

		// decode the current index
		var index bson.D
		err := cursor.Decode(&index)
		if err != nil {
			return err
		}

		// is the index empty
		if len(index) > 0 {

			// does index contain a key
			key := index.Map()["key"]

			if key != nil {
				// does the key contain ttl
				if key.(bson.D).Map()["ttl"] != nil {
					foundTTLIndex = true
				}
			}
		}
	}

	//https://docs.mongodb.com/manual/core/index-ttl/
	//
	// TTL indexes are special single-field indexes that MongoDB can use to automatically
	// remove documents from a collection after a certain amount of time or at a specific
	// clock time. Data expiration is useful for certain types of information like machine
	// generated event data, logs, and session information that only need to persist in a
	// database for a finite amount of time.
	//
	// To create a TTL index, use the db.collection.createIndex() method with the
	// expireAfterSeconds option on a field whose value is either a date or an array that
	// contains date values.
	//
	// TTL indexes expire documents after the specified number of seconds has passed since
	// the indexed field value; i.e. the expiration threshold is the indexed field value
	// plus the specified number of seconds.
	//
	// The _id field does not support TTL indexes.
	if !foundTTLIndex {
		_, err = s.Collection.Indexes().CreateOne(
			s.Context,
			mongo.IndexModel{
				Keys: bsonx.Doc{
					bsonx.Elem{
						Key:   "ttl",
						Value: bsonx.Int32(1),
					},
				},
				Options: options.Index().
					SetBackground(true).
					SetSparse(true).
					SetExpireAfterSeconds(int32(s.defaultCookie.MaxAge)),
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *MongoStore) findOne(session *gsessions.Session) error {
	// get the mongo _id from the cookie
	oid, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return err
	}

	// initialize an empty struct for FindOne to fill
	mongoSession := &MongoSession{}

	// find the session in mongo using the _id and put the result in the empty struct
	err = s.Collection.FindOne(
		s.Context,
		bson.M{
			"_id": oid,
		},
	).Decode(mongoSession)

	// no session found
	if errors.Is(err, mongo.ErrNoDocuments) {
		return fmt.Errorf("MongoStore: no session found: %w", err)
	}

	// something went wrong with the mongo search
	if err != nil {
		return fmt.Errorf("MongoStore: can't find session: %w", err)
	}

	// fill session.Values from mongo
	for k, v := range mongoSession.Data {
		session.Values[k] = v
	}

	return nil
}

func (s *MongoStore) insertOne(session *gsessions.Session) (*mongo.InsertOneResult, error) {
	// initialize a mongo session to insert
	mongoSession := &MongoSession{
		Data:     make(map[string]interface{}, len(session.Values)),
		Modified: primitive.NewDateTimeFromTime(time.Now()),
		Expires:  primitive.NewDateTimeFromTime(time.Now().Add(time.Duration(s.defaultCookie.MaxAge) * time.Second)),
		TTL:      primitive.NewDateTimeFromTime(time.Now()),
	}

	// get current session.Values
	for k, v := range session.Values {
		mongoSession.Data[k.(string)] = v
	}

	// insert the mongo session
	res, err := s.Collection.InsertOne(
		s.Context,
		mongoSession,
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *MongoStore) updateOne(session *gsessions.Session) (*mongo.UpdateResult, error) {
	// get the mongo _id from the cookie
	oid, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return nil, err
	}

	// initialize a mongo session to insert
	mongoSession := &MongoSession{
		Data:     make(map[string]interface{}, len(session.Values)),
		Modified: primitive.NewDateTimeFromTime(time.Now()),
		Expires:  primitive.NewDateTimeFromTime(time.Now().Add(time.Duration(s.defaultCookie.MaxAge) * time.Second)),
		TTL:      primitive.NewDateTimeFromTime(time.Now()),
	}

	// get current session.Values
	for k, v := range session.Values {
		mongoSession.Data[k.(string)] = v
	}

	// update session.Values in mongo usig the object id
	res, err := s.Collection.UpdateOne(
		s.Context,
		bson.M{
			"_id": oid,
		},
		bson.M{
			"$set": mongoSession,
		},
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *MongoStore) deleteOne(session *gsessions.Session) (*mongo.DeleteResult, error) {
	// convert session id to a mongo object id
	oid, err := primitive.ObjectIDFromHex(session.ID)
	if err != nil {
		return nil, err
	}

	// delete session using the object id
	res, err := s.Collection.DeleteOne(
		s.Context,
		bson.M{
			"_id": oid,
		},
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
