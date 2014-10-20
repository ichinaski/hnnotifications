package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"net/url"
	"strconv"
	"text/template"
)

const (
	host = "http://localhost:3000" // TODO: Don't hard-code the scheme

	errInvalidEmail = "Error: The email address is not valid!"
	errInvalidScore = "Error: The score field must be a number!"
	errInvalidLink  = "Error: The link is not valid."
	errNotFound     = "Error: The email address you provided is not subscribed to this service!"
	linkSentMsg     = "An account verification email has been sent."
	subscribedMsg   = "Your account is now active!"
	unsubscribedMsg = "You have been successfully unsubscribed."
)

var (
	router    *mux.Router
	templates = template.Must(template.ParseFiles(
		"templates/info.html",
		"templates/error.html",
		"templates/unsubscribe.html",
		"templates/item_email.html",
		"templates/activate_email.html",
		"templates/unsubscribe_email.html",
	))
)

// setupHandlers registers the http handlers of the app
func setupHandlers() {
	router = mux.NewRouter()
	router.HandleFunc("/subscribe", SubscribeHandler).
		Methods("POST")
	router.HandleFunc("/activate", ActivateHandler).
		Methods("GET").
		Name("activate")
	router.HandleFunc("/unsubscribe", UnsubscribeHandler).
		Name("unsubscribe")

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	http.Handle("/", router)
}

// SubscribeHandler will handle new registrations and score update requests
func SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	email, ok := parseEmail(r)
	if !ok {
		writeMessage(errInvalidEmail, w)
		return
	}
	score, ok := parseScore(r)
	if !ok {
		writeMessage(errInvalidScore, w)
		return
	}

	q := url.Values{} // Link query parameters
	u, ok := db.findUser(email)
	if ok {
		// The user already exists. The score will be added to the query
		q.Set("score", strconv.Itoa(score))
		u.Token = newToken() // reset user's token
		if err := db.updateToken(u.Id, u.Token); err != nil {
			writeError(err, w)
			return
		}
	} else {
		u = newUser(email, score)
		if err := db.upsertUser(u); err != nil {
			writeError(err, w)
			return
		}
	}

	q.Set("uid", u.Id.Hex())
	q.Set("t", u.Token)
	link := host + "/activate?" + q.Encode()
	go sendVerification(email, link)

	writeMessage(linkSentMsg, w)
}

func ActivateHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Display different messages on register/update
	uid, t := r.FormValue("uid"), r.FormValue("t")
	score, ok := parseScore(r)
	if ok {
		// We need to update the score too
		ok = db.updateScore(uid, t, score)
	} else {
		ok = db.activate(uid, t)
	}

	if ok {
		writeMessage(subscribedMsg, w)
	} else {
		writeMessage(errInvalidLink, w)
	}
}

// UnsubscribeHandler will handle unsubscription requests through a POST method,
// and will unsubscribe a user through a GET method and the user token
func UnsubscribeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		email, ok := parseEmail(r)
		u, found := db.findUser(email)
		if !ok || !found {
			writeMessage(errNotFound, w)
			return
		}

		u.Token = newToken() // reset user's token
		if err := db.updateToken(u.Id, u.Token); err != nil {
			writeError(err, w)
			return
		}

		q := url.Values{}
		q.Set("uid", u.Id.Hex())
		q.Set("t", u.Token)
		link := host + "/unsubscribe?" + q.Encode()
		go sendUnsubscription(email, link)

		writeMessage(linkSentMsg, w)
	case "GET":
		// If the user and token is provided in the query, unsubscribe the user.
		// Otherwise, display the unsubscription form
		uid, t := r.FormValue("uid"), r.FormValue("t")
		if uid != "" && t != "" {
			if db.unsubscribe(uid, t) {
				writeMessage(unsubscribedMsg, w)
			} else {
				writeMessage(errInvalidLink, w)
			}
		} else {
			if err := templates.ExecuteTemplate(w, "unsubscribe.html", nil); err != nil {
				writeError(err, w)
			}
		}
	}
}

func writeMessage(msg string, w http.ResponseWriter) {
	if err := templates.ExecuteTemplate(w, "info.html", msg); err != nil {
		writeError(err, w)
	}
}

// writeError renders the error in the HTTP response.
func writeError(err error, w http.ResponseWriter) {
	Logger.Println("Error: %v", err)
	w.WriteHeader(http.StatusInternalServerError)
	msg := "Oops! An error ocurred."
	if debug {
		msg = msg + " -  " + err.Error()
	}
	writeMessage(msg, w)
}
