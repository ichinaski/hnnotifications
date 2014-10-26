package main

import (
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"net/url"
	"strconv"
)

const (
	host = "http://localhost:3000" // TODO: Don't hard-code the scheme

	linkSentMsg     = "An account verification email has been sent."
	subscribedMsg   = "Your account is now active!"
	unsubscribedMsg = "You have been successfully unsubscribed."
)

var (
	errInvalidEmail = errors.New("Error: The email address is not valid!")
	errInvalidScore = errors.New("Error: The score field must be a number!")
	errInvalidLink  = errors.New("Error: The link is not valid.")
	errNotFound     = errors.New("Error: The email address you provided is not subscribed to this service!")
)

// errInternal represents an internal server error
type errInternal struct{ error }

// errMessage represents a meaningful error message, that will be sent to the user
type errMessage struct{ error }

type Context struct {
	db *Database
}

func newContext() *Context {
	return &Context{
		db: newDatabase(),
	}
}

func handler(f func(ctx *Context, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		Logger.Printf("%s %s\n", r.Method, r.URL.Path)
		ctx := newContext()
		defer ctx.db.close()

		err := f(ctx, w, r)

		if err == nil {
			return
		}

		// Log the error, and depending on the type, display it to the user
		Logger.Println(err)
		switch err.(type) {
		case errMessage:
			w.WriteHeader(http.StatusBadRequest)
			writeMessage(err.Error(), w)
		case errInternal:
		default:
			w.WriteHeader(http.StatusInternalServerError)
			writeMessage("Oops! An error ocurred.", w)
		}
	}
}

// setupHandlers registers the http handlers of the app
func setupHandlers() {
	router := mux.NewRouter()
	//router.HandleFunc("/", handler(IndexHandler)).  Methods("GET")
	router.HandleFunc("/subscribe", handler(SubscribeHandler)).
		Methods("POST")
	router.HandleFunc("/activate", handler(ActivateHandler)).
		Methods("GET")
	router.HandleFunc("/unsubscribe", handler(UnsubscribeHandler)).
		Methods("GET", "POST")

	// serve settings.html static file. index.html works the same way,
	// though it's automatically handled by the root handler
	router.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/settings.html")
	})

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	http.Handle("/", router)
}

// SubscribeHandler will handle new registrations and score update requests
func SubscribeHandler(ctx *Context, w http.ResponseWriter, r *http.Request) error {
	email, ok := parseEmail(r)
	if !ok {
		return errMessage{errInvalidEmail}
	}
	score, ok := parseScore(r)
	if !ok {
		return errMessage{errInvalidScore}
	}

	q := url.Values{} // Link query parameters
	u, ok := ctx.db.findUser(email)
	if ok {
		// The user already exists. The score will be added to the query
		q.Set("score", strconv.Itoa(score))
		u.Token = newToken() // reset user's token
		if err := ctx.db.updateToken(u.Id, u.Token); err != nil {
			return errInternal{err}
		}
	} else {
		u = newUser(email, score)
		if err := ctx.db.upsertUser(u); err != nil {
			return errInternal{err}
		}
	}

	q.Set("uid", u.Id.Hex())
	q.Set("t", u.Token)
	link := host + "/activate?" + q.Encode()
	go sendVerification(email, link)

	return writeMessage(linkSentMsg, w)
}

func ActivateHandler(ctx *Context, w http.ResponseWriter, r *http.Request) error {
	// TODO: Display different messages on register/update
	uid, t := r.FormValue("uid"), r.FormValue("t")
	score, ok := parseScore(r)
	if ok {
		// We need to update the score too
		ok = ctx.db.updateScore(uid, t, score)
	} else {
		ok = ctx.db.activate(uid, t)
	}

	if ok {
		return writeMessage(subscribedMsg, w)
	}
	return errMessage{errInvalidLink}
}

// UnsubscribeHandler will handle unsubscription requests through a POST method,
// and will unsubscribe a user through a GET method and the user token
func UnsubscribeHandler(ctx *Context, w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "POST":
		email, ok := parseEmail(r)
		u, found := ctx.db.findUser(email)
		if !ok || !found {
			return errMessage{errNotFound}
		}

		u.Token = newToken() // reset user's token
		if err := ctx.db.updateToken(u.Id, u.Token); err != nil {
			return errInternal{err}
		}

		q := url.Values{}
		q.Set("uid", u.Id.Hex())
		q.Set("t", u.Token)
		link := host + "/unsubscribe?" + q.Encode()
		go sendUnsubscription(email, link)

		return writeMessage(linkSentMsg, w)
	case "GET":
		uid, t := r.FormValue("uid"), r.FormValue("t")
		if ctx.db.unsubscribe(uid, t) {
			return writeMessage(unsubscribedMsg, w)
		}
		return errMessage{errInvalidLink}
	}
	return nil
}

// writeMessage renders a message in the default 'info' template
func writeMessage(msg string, w http.ResponseWriter) error {
	return useTemplate("info", msg, w)
}
