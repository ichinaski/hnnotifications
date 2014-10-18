package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Item represents a HN story
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

// start sets up a goroutine that will periodically call run()
func initNotifier() {
	go func() {
		ticker := time.NewTicker(runInterval)
		for {
			select {
			case <-ticker.C:
				runNotifier() // TODO: Should we call process on a separate goroutine?
			}
		}
	}()
}

// runNotifier fetches the top HN stories and sends notifications according to each user's score threshold
func runNotifier() {
	Logger.Println("Running...")
	ids := getTopStories()

	// fetcher runs a goroutine to fetch the item. Once completed, the result
	// will be inserted in the returned channel, and the channel closed.
	fetchItem := func(id int) chan Item {
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

	// Fetch items concurrently. Each channel will only hold one item
	cs := make([]chan Item, len(ids))
	for i, id := range ids {
		cs[i] = fetchItem(id)
	}

	for item := range merge(cs...) {
		if item.Id == 0 {
			Logger.Println("Error: We received an empty item")
			continue

		}

		for _, u := range db.FindUsersForItem(item.Id, item.Score) {
			// skip the user if the account is not activated yet
			if !u.Active {
				continue
			}

			err := sendItem(item.Id, item.Title, item.Url, u.Email)
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

	Logger.Println("Notifier finished")
}

// getTopStories reads the top stories IDs from the API
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

// getItem reads the HN story item from the API
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
