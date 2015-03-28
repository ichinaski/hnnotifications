package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	topStoriesUrl = "https://hacker-news.firebaseio.com/v0/topstories.json"
	itemUrl       = "https://hacker-news.firebaseio.com/v0/item/%d.json"

	runInterval   = 15 * time.Minute // Interval at which we fetch the items
	maxTopStories = 150              // Maximum top stories fetched per cycle
)

var (
	config = loadConfig()
	Logger = log.New(os.Stdout, "  ", log.LstdFlags|log.Lshortfile)
)

func main() {
	initDb() // Will panic on failure
	setupHandlers()

	// set up a goroutine that will periodically call run()
	go func() {
		run()
		ticker := time.NewTicker(runInterval)
		for {
			select {
			case <-ticker.C:
				run() // TODO: Consider executing run() in a separate goroutine
			}
		}
	}()

	Logger.Printf("Listening on %s...\n", config.Addr)
	Logger.Fatal(http.ListenAndServe(config.Addr, nil))
}

// Item represents a HN story.
type Item struct {
	By    string // unused
	Id    int
	Kids  []int64 // unused
	Score int
	Time  int64 // unused
	Title string
	Type  string // unused
	Url   string
}

// run fetches the top HN stories and sends notifications according to each user's score threshold.
// The channel fan-in approach is fully inspired by the example in http://blog.golang.org/pipelines.
func run() {
	Logger.Println("Notifier started...")
	t0 := time.Now()
	db := newDatabase()
	defer db.close()

	// Single http.Client concurrently used by all goroutines
	client := &http.Client{}

	ids, err := getTopStories(client)
	if err != nil {
		Logger.Println(err)
		return // Just wait till the next cycle.
	}
	if len(ids) > maxTopStories {
		ids = ids[:maxTopStories]
	}

	// fetcher runs a goroutine to fetch the item. Once completed, the result
	// will be inserted in the returned channel, and the channel closed.
	fetchItem := func(id int) chan Item {
		out := make(chan Item)
		go func() {
			item, err := getItem(client, id)
			if err != nil {
				Logger.Println(err)
			} else if id == item.Id { // Guard against null responses, coerced into an empty Item
				out <- item
			}
			close(out)
		}()
		return out
	}

	// Fetch items concurrently. Each channel will just hold one item.
	cs := make([]chan Item, len(ids))
	for i, id := range ids {
		cs[i] = fetchItem(id)
	}

	for item := range merge(cs...) {
		if users := db.findUsersForItem(item.Id, item.Score); len(users) > 0 {
			emails := make([]string, len(users)) // Create a slice with all the recipients for this item.
			for i, u := range users {
				emails[i] = u.Email
			}

			// Send the email.
			if err := sendItem(item.Id, item.Title, item.Url, emails); err != nil {
				Logger.Println("Error sending mail: ", err)
				continue
			}
			Logger.Printf("Item %d sent to users: %v\n", item.Id, emails)

			// Update items set.
			if err := db.updateSentItems(emails, item.Id); err != nil {
				Logger.Println("Error: updateItems() - ", err)
			}
		}
	}

	Logger.Printf("Notifier finished - Total time: %s\n", time.Now().Sub(t0).String())
}

// getTopStories reads the top stories IDs from the API.
func getTopStories(client *http.Client) ([]int, error) {
	req, err := http.NewRequest("GET", topStoriesUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Close = true

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	res := struct {
		Items []int
	}{}

	err = json.NewDecoder(resp.Body).Decode(&res.Items)
	return res.Items, err
}

// getItem reads the HN story item from the API.
func getItem(client *http.Client, id int) (item Item, err error) {
	var req *http.Request
	req, err = http.NewRequest("GET", fmt.Sprintf(itemUrl, id), nil)
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
	return item, err
}

// merge converts a list of channels to a single channel.
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
