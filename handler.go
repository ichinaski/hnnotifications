package main

import (
	"errors"
	"github.com/gorilla/mux"
	"labix.org/v2/mgo"
	"net/http"
	"net/url"
	"strconv"
	"text/template"
)

const (
	host = "http://localhost:8080" // TODO: Don't hard-code the scheme
)

var (
	errInvalidLink = errors.New("Error: The link is not valid.")

	router *mux.Router

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

// SubscribeHandler will handle new registrations through a POST method,
// and email verification though a GET method and the user token
func SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	threshold, err := strconv.Atoi(r.FormValue("threshold"))
	if err != nil {
		writeError(err, w)
		return
	}

	u := newUser(email, threshold)
	if err := db.upsertUser(u); err != nil {
		if mgo.IsDup(err) {
			writeMessage("This email account is already subscribed!", w)
		} else {
			writeError(err, w)
		}
		return
	}

	path, _ := router.Get("activate").URL()
	q := url.Values{}
	q.Set("uid", u.Id.Hex())
	q.Set("t", u.Token)
	href := host + path.String() + "?" + q.Encode()
	go sendVerification(email, href)

	writeMessage("An account verification email has been sent.", w)
}

// UnsubscribeHandler will handle unsubscription requests through a POST method,
// and will unsubscribe a user through a GET method and the user token
func UnsubscribeHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		email := r.FormValue("email")
		if email == "" {
			writeMessage("You must provide a valid email!", w)
			return
		}

		u, err := db.findUser(email)
		if err != nil {
			if err == mgo.ErrNotFound {
				writeMessage("The email address you provided is not subscribed to this service!", w)
			} else {
				writeError(err, w)
			}
			return

		}

		u.Token = newToken() // reset user's token
		if err := db.updateToken(u.Id, u.Token); err != nil {
			writeError(err, w)
			return
		}

		href := "http://" + r.Host // TODO: Do not hard-code the scheme!
		path, _ := router.Get("unsubscribe").URL()
		q := url.Values{}
		q.Set("uid", u.Id.Hex())
		q.Set("t", u.Token)
		href = href + path.String() + "?" + q.Encode()
		go sendUnsubscription(email, href)

		writeMessage("The unsubscribe link has been sent to your email.", w)
	case "GET":
		// If the user and token is provided in the query, unsubscribe the user.
		// Otherwise, display the unsubscription form
		uid, t := r.FormValue("uid"), r.FormValue("t")
		if uid != "" && t != "" {
			if db.unsubscribe(uid, t) {
				writeMessage("You have been successfully unsubscribed.", w)
			} else {
				writeError(errInvalidLink, w)
			}
		} else {
			if err := templates.ExecuteTemplate(w, "unsubscribe.html", nil); err != nil {
				writeError(err, w)
			}
		}
	}
}

func ActivateHandler(w http.ResponseWriter, r *http.Request) {
	uid, t := r.FormValue("uid"), r.FormValue("t")
	if db.activate(uid, t) {
		writeMessage("Your account is now active!", w)
	} else {
		writeError(errInvalidLink, w)
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
