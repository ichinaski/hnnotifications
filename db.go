package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

var (
	session *mgo.Session // Though global, this session is meant to be copied for each database object creation
)

// initDb sets up the DB configuration. Panics upon error
func initDb() {
	var err error
	session, err = mgo.Dial(config.DBAddr)
	if err != nil {
		panic(err)
	}
	Logger.Println("Connected to MongoDB")

	session.EnsureSafe(&mgo.Safe{})
	// mgo.SetLogger(Logger)
	// mgo.SetDebug(true)

	db := newDatabase()
	defer db.close()

	//create an index for the email field on the users collection
	if err := db.users.EnsureIndex(mgo.Index{
		Key:    []string{"email"},
		Unique: true,
	}); err != nil {
		panic(err)
	}

	// create score index
	if err := db.users.EnsureIndex(mgo.Index{
		Key: []string{"score"},
	}); err != nil {
		panic(err)
	}

	if err := db.users.EnsureIndex(mgo.Index{
		Key: []string{"score", "sentItems", "active"},
	}); err != nil {
		panic(err)
	}
}

type User struct {
	Id        bson.ObjectId `bson:"_id"`       // Unique Identifier
	Email     string        `bson:"email"`     // User mail. We do not need any more details
	Score     int           `bson:"score"`     // Minimum score for an item to be sent
	SentItems []int         `bson:"sentItems"` // Sent item ids
	Token     string        `bson:"token"`     // User token
	Active    bool          `bson:"active"`
	CreatedAt time.Time     `bson:"createdAt"` // Registration time
	// TODO: Add queue for unprocessed items (batch notifications)
}

func newUser(email string, score int) *User {
	return &User{
		Id:        bson.NewObjectId(),
		Email:     email,
		Score:     score,
		Token:     newToken(),
		Active:    false, // Email verification required
		CreatedAt: time.Now(),
	}
}

type Database struct {
	mdb   *mgo.Database
	users *mgo.Collection
}

// newDatabase created a new Database, cloning the initial mgo.Session
// The caller *must* call close() before disposing the Database
func newDatabase() *Database {
	s := session.Copy()
	mdb := s.DB("hnnotifications")
	return &Database{
		mdb:   mdb,
		users: mdb.C("users"),
	}
}

func (db *Database) close() {
	db.mdb.Session.Close()
}

func (db *Database) upsertUser(u *User) (err error) {
	_, err = db.users.UpsertId(u.Id, u)
	return
}

func (db *Database) validate(email, token string) (*User, bool) {
	if email == "" || token == "" {
		Logger.Printf("User validation error: %s - %s\n", email, token)
		return nil, false
	}

	var u User
	if err := db.users.Find(bson.M{"email": email, "token": token}).One(&u); err != nil {
		Logger.Printf("User validation error: %s - %s. %v\n", email, token, err)
		return nil, false
	}
	return &u, true
}

func (db *Database) activate(email, token string) bool {
	u, ok := db.validate(email, token)
	if !ok {
		return false
	}

	update := bson.M{
		"$set": bson.M{
			"active": true,
			"token":  nil,
		},
	}
	err := db.users.UpdateId(u.Id, update)
	if err != nil {
		Logger.Println("Error: activate() - ", err)
	}
	return err == nil
}

func (db *Database) unsubscribe(email, token string) bool {
	u, ok := db.validate(email, token)
	if !ok {
		return false
	}

	err := db.users.RemoveId(u.Id)
	if err == nil {
		return true
	}
	Logger.Println("Error: unsubscribe() - ", err)
	return false
}

// updateScore validates the user and updates the score threshold
func (db *Database) updateScore(email, token string, score int) bool {
	u, ok := db.validate(email, token)
	if !ok {
		return false
	}

	update := bson.M{
		"$set": bson.M{
			"score":  score,
			"token":  nil,
			"active": true,
		},
	}
	err := db.users.UpdateId(u.Id, update)
	if err != nil {
		Logger.Println("Error: updateScore() - ", err)
	}
	return err == nil
}

func (db *Database) findUsersForItem(item, score int) []User {
	var result []User
	err := db.users.Find(bson.M{"score": bson.M{"$lte": score}, "sentItems": bson.M{"$ne": item}, "active": true}).All(&result)
	if err != nil {
		Logger.Println(err)
	}

	return result
}

// UpdateSentItems adds the given item to the sentItems set in the user object
func (db *Database) updateSentItems(uid bson.ObjectId, item int) error {
	update := bson.M{
		"$addToSet": bson.M{
			"sentItems": item,
		},
	}

	return db.users.UpdateId(uid, update)
}

func (db *Database) updateToken(uid bson.ObjectId, token string) error {
	update := bson.M{
		"$set": bson.M{
			"token": token,
		},
	}
	return db.users.UpdateId(uid, update)
}

func (db *Database) findUser(email string) (*User, bool) {
	var u User
	err := db.users.Find(bson.M{"email": email}).One(&u)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Println("Error: findUser() - ", err)
	}
	return &u, err == nil
}
