package main

import (
	"github.com/gorilla/mux"
	"labix.org/v2/mgo"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"text/template"
	"time"
)

const (
	topStoriesUrl = "https://hacker-news.firebaseio.com/v0/topstories.json"
	itemUrl       = "https://hacker-news.firebaseio.com/v0/item/%d.json"
	commentsUrl   = "https://news.ycombinator.com/item?id=%d"

	runInterval = 15 * time.Minute // Interval at which we fetch the items
)

var (
	//hnUsername = "hnnotifications"
	//hnPassword = "f(dY4Bx_9U"
	//hnSmtpAddr = "smtp.gmail.com:587"
	//hnEmail = "hnnotifications@gmail.com"

	// TODO: Shitfuck use env vars instead!!
	hnUsername = "inigo@ichinaski.com"
	hnPassword = "8gvBue45"
	hnSmtpHost = "smtp.zoho.com"
	hnSmtpAddr = "smtp.zoho.com:587"
	hnEmail    = "inigo@ichinaski.com"
)

var (
	debug  = true
	db     *Database
	Logger = log.New(os.Stdout, "  ", log.LstdFlags|log.Lshortfile)
	router *mux.Router
)

var (
	templates = template.Must(template.ParseFiles(
		"templates/info.html",
		"templates/error.html",
		"templates/item.html",
		"templates/activate.html",
	))
)

func main() {
	var err error
	db, err = CreateDB()
	if err != nil {
		Logger.Fatal(err)
	}

	initNotifier() // start the notification system

	// Set up handlers
	router = mux.NewRouter()
	router.HandleFunc("/subscribe", SubscribeHandler).
		Methods("POST")
	router.HandleFunc("/activate", ActivateHandler).
		Methods("GET").
		Name("activate")

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/")))
	http.Handle("/", router)
	Logger.Println("Listening...")
	http.ListenAndServe(":8080", nil)
}

// SubscribeHandler will handle new registrations through a POST method,
// and email verification though a GET method and the user token
func SubscribeHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	threshold, err := strconv.Atoi(r.FormValue("threshold"))
	if err != nil {
		writeError(w, err)
		return
	}

	u := NewUser(email, threshold)
	if err := db.UpsertUser(u); err != nil {
		if mgo.IsDup(err) {
			writeMessage("This email account is already subscribed!", w)
		} else {
			writeError(w, err)
		}
		return
	}

	href := "http://" + r.Host // TODO: Do not hard-code the scheme!
	path, _ := router.Get("activate").URL()
	q := url.Values{}
	q.Set("uid", u.Id.Hex())
	q.Set("t", u.Token)
	href = href + path.String() + "?" + q.Encode()
	go sendVerification(email, href)

	writeMessage("An account verification email has been sent.", w)
}

func ActivateHandler(w http.ResponseWriter, r *http.Request) {
	uid, t := r.FormValue("uid"), r.FormValue("t")
	if db.Activate(uid, t) {
		writeMessage("Your account is now active!", w)
	} else {
		writeMessage("Error. The link is not valid", w)
	}
}

func writeMessage(msg string, w http.ResponseWriter) {
	if err := templates.ExecuteTemplate(w, "info.html", msg); err != nil {
		writeError(w, err)
	}
}

// writeError renders the error in the HTTP response.
func writeError(w http.ResponseWriter, err error) {
	Logger.Println("Error: %v", err)
	w.WriteHeader(http.StatusInternalServerError)
	msg := "Oops! An error ocurred."
	if debug {
		msg = msg + " -  " + err.Error()
	}
	writeMessage(msg, w)
}
