package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"sync"
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

	hnUsername = "inigo@ichinaski.com"
	hnPassword = "8gvBue45"
	hnSmtpHost = "smtp.zoho.com"
	hnSmtpAddr = "smtp.zoho.com:587"
	hnEmail    = "inigo@ichinaski.com"
)

var (
	db     *Database
	Logger = log.New(os.Stdout, "  ", log.LstdFlags|log.Lshortfile)
)

type Item struct {
	By    string
	Id    int
	Kids  []int64
	Score int
	Time  int64
	Title string
	Type  string
	Url   string
}

func main() {
	var err error
	db, err = CreateDB()
	if err != nil {
		Logger.Fatal(err)
	}

	// Handle the notifications system in a separate goroutine
	go func() {
		run()
		ticker := time.NewTicker(runInterval)
		for {
			select {
			case <-ticker.C:
				run() // TODO: Should we call process on a separate goroutine?
			}
		}
	}()

	Logger.Println("Listening...")
	http.ListenAndServe(":8080", nil)
}

func run() {
	Logger.Println("Running...")
	ids := getTopStories()

	// fetcher runs a goroutine to fetch the item. Once completed, the result
	// will be inserted in the returned channel, and the channel closed.
	fetcher := func(id int) chan Item {
		out := make(chan Item)
		go func() {
			item, err := getItem(id)
			if err != nil {
				Logger.Println(err)
			} else {
				out <- item
			}
			close(out)
		}()
		return out
	}

	cs := make([]chan Item, len(ids))
	for i, id := range ids {
		// Fetch items concurrently. Each channel will only hold one item
		cs[i] = fetcher(id)
	}

	items := merge(cs...)

	processItems(items)
	Logger.Println("Fetching and notifying cycle finished")
}

func processItems(items <-chan Item) {
	for item := range items {
		if item.Id == 0 {
			Logger.Println("processItems() - We received an empty item")
			continue

		}
		for _, u := range db.FindUsersForItem(item.Id, item.Score) {
			err := sendMail(item.Id, item.Title, item.Url, u.Email)
			if err != nil {
				Logger.Println("Error sending mail: ", err)
				continue
			}
			Logger.Printf("Item %d sent to user %s\n", item.Id, u.Email)
			if err := db.UpdateSentItems(u.Id, item.Id); err != nil {
				Logger.Println(err)
			}
		}
	}
}

func getTopStories() []int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", topStoriesUrl, nil)
	if err != nil {
		Logger.Fatal(err)
	}
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("%s", err))
	}
	defer resp.Body.Close()

	res := struct {
		Items []int
	}{}

	err = json.NewDecoder(resp.Body).Decode(&res.Items)
	if err != nil {
		panic(fmt.Sprintf("%s", err))
	}

	return res.Items
}

func getItem(id int) (item Item, err error) {
	url := fmt.Sprintf(itemUrl, id)
	client := &http.Client{}
	var req *http.Request
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Close = true

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&item)
	return item, nil
}

func sendMail(id int, title, url, email string) error {
	// Set up authentication information.
	auth := smtp.PlainAuth("", hnUsername, hnPassword, hnSmtpHost)

	from := mail.Address{"HN Notifications", hnEmail}
	//to := mail.Address{"", email}
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = email
	headers["Subject"] = title
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"utf-8\""

	message := title + ": " + url +
		"\nHacker News discussion: " + fmt.Sprintf(commentsUrl, id) +
		"\n\n--\nHN Notifications"

	body := ""
	for k, v := range headers {
		body += fmt.Sprintf("%s: %s\n", k, v)
	}
	body += "\n" + message

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	return smtp.SendMail(hnSmtpAddr, auth, hnEmail, []string{email}, []byte(body))
}

// merge converts a list of channels to a single channel.
// Based on the example in http://blog.golang.org/pipelines
func merge(cs ...chan Item) <-chan Item {
	var wg sync.WaitGroup
	out := make(chan Item)

	// Start an output goroutine for each input channel in cs. output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan Item) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done. This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
