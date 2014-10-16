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
	CreatedAt time.Time     `bson:"createdAt"` // Registration time
	// TODO: Add queue for unprocessed items (batch notifications)
}

type Database struct {
	db        *mgo.Database
	usersColl *mgo.Collection
}

func CreateDB() (*Database, error) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		return nil, err
	}
	Logger.Println("Connected to MongoDB")
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
		Key: []string{"threshold", "sentItems"},
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

func (db *Database) FindUsersForItem(item, score int) []User {
	var result []User
	err := db.usersColl.Find(bson.M{"threshold": bson.M{"$lte": score}, "sentItems": bson.M{"$ne": item}}).All(&result)
	if err != nil {
		Logger.Println(err)
	}

	return result
}

// UpdateSentItems adds the given item to the sentItems set in the user object
func (db *Database) UpdateSentItems(uid bson.ObjectId, item int) error {
	update := bson.M{
		"$addToSet": bson.M{
			"sentItems": item,
		},
	}

	return db.usersColl.UpdateId(uid, update)
}
