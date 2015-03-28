package main

import (
	"time"

	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

var (
	session *mgo.Session // Though global, this session is meant to be copied for each database instance.
)

// initDb sets up the DB configuration. Panics upon error.
func initDb() {
	var err error
	session, err = mgo.Dial(config.DBAddr)
	if err != nil {
		panic(err)
	}
	Logger.Println("Connected to MongoDB")

	session.EnsureSafe(&mgo.Safe{})

	db := newDatabase()
	defer db.close()

	// ensure indexes
	if err := db.users.EnsureIndex(mgo.Index{
		Key:    []string{"email"},
		Unique: true,
	}); err != nil {
		panic(err)
	}

	if err := db.users.EnsureIndex(mgo.Index{
		Key:    []string{"email", "token"},
		Unique: true,
	}); err != nil {
		panic(err)
	}

	if err := db.users.EnsureIndex(mgo.Index{
		Key: []string{"score", "sentItems", "active"},
	}); err != nil {
		panic(err)
	}
}

// User represents a user subscribed to the service.
type User struct {
	Id        bson.ObjectId `bson:"_id"`       // Unique Identifier.
	Email     string        `bson:"email"`     // User mail. We do not need any more details.
	Score     int           `bson:"score"`     // Minimum score for an item to be sent.
	Keywords  []string      `bson:"keywords"`  // Keywords 'subscribed' to
	SentItems []int         `bson:"sentItems"` // Sent item ids.
	Token     string        `bson:"token"`     // User token.
	Active    bool          `bson:"active"`    // Account status.
	CreatedAt time.Time     `bson:"createdAt"` // Registration time.
	// TODO: Add queue for unprocessed items (batch notifications).
}

// newUser creates a new user, with a randomly generated token.
func newUser(email string, score int, keywords []string) *User {
	return &User{
		Id:        bson.NewObjectId(),
		Email:     email,
		Score:     score,
		Keywords:  keywords,
		Token:     newToken(),
		Active:    false, // Email verification required.
		CreatedAt: time.Now(),
	}
}

// Database is a convenient struct to wrap mgo collection(s).
type Database struct {
	mdb   *mgo.Database
	users *mgo.Collection
}

// newDatabase created a new Database, cloning the initial mgo.Session.
// The caller *must* call close() before disposing the Database.
func newDatabase() *Database {
	s := session.Copy()
	mdb := s.DB("hnnotifications")
	return &Database{
		mdb:   mdb,
		users: mdb.C("users"),
	}
}

// close handles the underlying session closure.
func (db *Database) close() {
	db.mdb.Session.Close()
}

// upsertUser inserts/updates a user into the database.
func (db *Database) upsertUser(u *User) (err error) {
	_, err = db.users.UpsertId(u.Id, u)
	return
}

// validate checks whether the user and token pair is valid, returning the user if found.
func (db *Database) validate(email, token string) *User {
	if email == "" || token == "" {
		Logger.Printf("User validation error: %s - %s\n", email, token)
		return nil
	}

	var u User
	if err := db.users.Find(bson.M{"email": email, "token": token}).One(&u); err != nil {
		Logger.Printf("User validation error: %s - %s. %v\n", email, token, err)
		return nil
	}
	return &u
}

// activate sets the account status to 'active'.
func (db *Database) activate(email, token string) bool {
	u := db.validate(email, token)
	if u == nil {
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

// unsubscribe completely removes the user account from the database.
func (db *Database) unsubscribe(email, token string) bool {
	u := db.validate(email, token)
	if u == nil {
		return false
	}

	err := db.users.RemoveId(u.Id)
	if err == nil {
		return true
	}
	Logger.Println("Error: unsubscribe() - ", err)
	return false
}

// updateScore validates the user and updates the score threshold.
func (db *Database) updateUser(email, token string, score int, keywords []string) bool {
	u := db.validate(email, token)
	if u == nil {
		return false
	}

	update := bson.M{
		"$set": bson.M{
			"score":    score,
			"keywords": keywords,
			"token":    nil,
			"active":   true,
		},
	}
	err := db.users.UpdateId(u.Id, update)
	if err != nil {
		Logger.Println("Error: updateUser() - ", err)
	}
	return err == nil
}

// findUsersForItem queries all users entitled to receive a given item.
func (db *Database) findUsersForItem(item Item) []User {
	query := bson.M{
		"score":     bson.M{"$lte": item.Score},
		"sentItems": bson.M{"$ne": item.Id},
		"active":    true,
		"$or": []bson.M{
			bson.M{"keywords": bson.M{"$exists": false}},
			bson.M{"keywords": bson.M{"$size": 0}},
			bson.M{"keywords": bson.M{"$in": Keywords(item.Title)}},
		},
	}

	var result []User
	err := db.users.Find(query).All(&result)
	if err != nil {
		Logger.Println(err)
	}

	return result
}

// updateSentItems adds the given item to each user's item set.
func (db *Database) updateSentItems(emails []string, item int) error {
	selector := bson.M{"email": bson.M{"$in": emails}}

	update := bson.M{
		"$addToSet": bson.M{
			"sentItems": item,
		},
	}

	_, err := db.users.UpdateAll(selector, update)
	return err
}

// updateToken assigns the new token to the user.
func (db *Database) updateToken(uid bson.ObjectId, token string) error {
	update := bson.M{
		"$set": bson.M{
			"token": token,
		},
	}
	return db.users.UpdateId(uid, update)
}

// findUser queries a user by its email field.
func (db *Database) findUser(email string) (*User, bool) {
	var u User
	err := db.users.Find(bson.M{"email": email}).One(&u)
	if err != nil && err != mgo.ErrNotFound {
		Logger.Println("Error: findUser() - ", err)
	}
	return &u, err == nil
}
