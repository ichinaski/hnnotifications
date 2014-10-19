package main

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

type User struct {
	Id        bson.ObjectId `bson:"_id"`       // Unique Identifier
	Email     string        `bson:"email"`     // User mail. We do not need any more details
	Threshold int           `bson:"threshold"` // Minimum score for an item to be sent
	SentItems []int         `bson:"sentItems"` // Sent item ids
	Token     string        `bson:"token"`     // User token
	Active    bool          `bson:"active"`
	CreatedAt time.Time     `bson:"createdAt"` // Registration time
	// TODO: Add queue for unprocessed items (batch notifications)
}

func newUser(email string, threshold int) *User {
	return &User{
		Id:        bson.NewObjectId(),
		Email:     email,
		Threshold: threshold,
		Token:     newToken(),
		Active:    false, // Email verification required
		CreatedAt: time.Now(),
	}
}

type Database struct {
	db        *mgo.Database
	usersColl *mgo.Collection
}

func setupDB() (*Database, error) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		return nil, err
	}
	Logger.Println("Connected to MongoDB")

	session.EnsureSafe(&mgo.Safe{})
	// mgo.SetLogger(Logger)
	// mgo.SetDebug(true)

	database := session.DB("hnnotifications")
	db := &Database{
		db:        database,
		usersColl: database.C("users"),
		// TODO: Create a new collection for sent notifications (ids, times, users, etc)
	}

	//create an index for the email field on the users collection
	if err := db.usersColl.EnsureIndex(mgo.Index{
		Key:    []string{"email"},
		Unique: true,
	}); err != nil {
		panic(err)
	}

	// create threshold index
	if err := db.usersColl.EnsureIndex(mgo.Index{
		Key: []string{"threshold"},
	}); err != nil {
		panic(err)
	}

	if err := db.usersColl.EnsureIndex(mgo.Index{
		Key: []string{"threshold", "sentItems", "active"},
	}); err != nil {
		panic(err)
	}

	// create sent items index
	/*
		// create sent items index
		if err := db.usersColl.EnsureIndex(mgo.Index{
			Key: []string{"email", "notifications.sentAt"},
		}); err != nil {
			panic(err)
		}
	*/

	//u := User{bson.NewObjectId(), "ichinaski", 400, []int{8441939, 8450147, 8448617}, time.Now()}
	//db.usersColl.UpsertId(u.Id, u)

	return db, nil
}

func (db *Database) upsertUser(u *User) (err error) {
	_, err = db.usersColl.UpsertId(u.Id, u)
	return
}

func (db *Database) validate(uid, token string) (*User, bool) {
	if uid == "" || token == "" || !bson.IsObjectIdHex(uid) {
		return nil, false
	}
	var u User
	err := db.usersColl.FindId(bson.ObjectIdHex(uid)).One(&u)
	if err != nil {
		Logger.Println("Error: validate() - ", err)
		return nil, false
	}

	return &u, token == u.Token
}

func (db *Database) activate(uid, token string) bool {
	u, ok := db.validate(uid, token)
	if !ok {
		return false
	}

	u.Active = true
	err := db.upsertUser(u)
	if err == nil {
		return true
	}
	Logger.Println("Error: activate() - ", err)
	return false
}

func (db *Database) unsubscribe(uid, token string) bool {
	u, ok := db.validate(uid, token)
	if !ok {
		return false
	}

	err := db.usersColl.RemoveId(u.Id)
	if err == nil {
		return true
	}
	Logger.Println("Error: unsubscribe() - ", err)
	return false
}

func (db *Database) findUsersForItem(item, score int) []User {
	var result []User
	err := db.usersColl.Find(bson.M{"threshold": bson.M{"$lte": score}, "sentItems": bson.M{"$ne": item}, "active": true}).All(&result)
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

	return db.usersColl.UpdateId(uid, update)
}

func (db *Database) updateToken(uid bson.ObjectId, token string) error {
	update := bson.M{
		"$set": bson.M{
			"token": token,
		},
	}
	return db.usersColl.UpdateId(uid, update)
}

func (db *Database) findUser(email string) (*User, error) {
	var u User
	err := db.usersColl.Find(bson.M{"email": email}).One(&u)
	return &u, err
}
