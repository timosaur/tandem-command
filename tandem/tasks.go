package tandem

import (
	"fmt"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
)

func init() {
	http.HandleFunc("/tasks/update", update)
	http.HandleFunc("/tasks/updateCommand", updateCommandHandler)
}

func LoadCredentials(c appengine.Context) (client *twittergo.Client, bot *Bot, err error) {
	bot = new(Bot)
	k := datastore.NewKey(c, "Bot", "", 1, nil)
	if err = datastore.Get(c, k, bot); err != nil {
		if err == datastore.ErrNoSuchEntity {
			if _, putErr := datastore.Put(c, k, bot); putErr != nil {
				err = putErr
				return
			}
		}
		return
	}
	config := &oauth1a.ClientConfig{
		ConsumerKey:    bot.ConsumerKey,
		ConsumerSecret: bot.ConsumerSecret,
	}
	user := oauth1a.NewAuthorizedConfig(bot.BotKey, bot.BotSecret)
	client = twittergo.NewClient(config, user)
	return
}

func update(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		client  *twittergo.Client
		bot     *Bot
		req     *http.Request
		resp    *twittergo.APIResponse
		max_id  uint64
		query   url.Values
		results *twittergo.Timeline
		updated time.Time
	)
	const (
		count   int = 200
		urltmpl     = "/1.1/statuses/mentions_timeline.json?%v"
	)

	c := appengine.NewContext(r)

	if client, bot, err = LoadCredentials(c); err != nil {
		fmt.Fprintf(w, "Error loading credentials: %v\n", err)
		return
	}
	defer datastore.Put(c, datastore.NewKey(c, "Bot", "", 1, nil), bot)

	query = url.Values{}
	query.Set("count", fmt.Sprintf("%v", count))
	query.Set("since_id", bot.LastUpdateId)

	latest_status := make(map[string]twittergo.Tweet)
	commands := make(map[int64]struct{})

	for {
		if max_id != 0 {
			query.Set("max_id", fmt.Sprintf("%v", max_id))
		}
		endpoint := fmt.Sprintf(urltmpl, query.Encode())
		if req, err = http.NewRequest("GET", endpoint, nil); err != nil {
			fmt.Fprintf(w, "Could not parse request: %v\n", err)
			return
		}
		if resp, err = client.SendRequest(req); err != nil {
			fmt.Fprintf(w, "Could not send request: %v\n", err)
			return
		}
		results = &twittergo.Timeline{}
		if err = resp.Parse(results); err != nil {
			if rle, ok := err.(twittergo.RateLimitError); ok {
				fmt.Fprintf(w, "Rate limited. Reset at %v\n", rle.Reset)
				return
			} else {
				fmt.Fprintf(w, "Problem parsing response: %v\n", err)
			}
		}
		batch := len(*results)
		if batch == 0 {
			fmt.Fprintf(w, "No more results, end of timeline.\n")
			break
		}

		for i, tweet := range *results {
			if i == 0 && updated.IsZero() {
				bot.LastUpdateId = tweet.IdStr()
				updated = tweet.CreatedAt()
			}
			id_str, err := strconv.Atoi(tweet.IdStr())
			if err != nil {
				continue
			}
			max_id = uint64(id_str)

			fmt.Fprintf(w, "tweet %d [%d] %s: %s\n", i, max_id, tweet.User().ScreenName(), tweet.Text())
			if _, exists := latest_status[tweet.User().ScreenName()]; exists {
				fmt.Fprintf(w, "\n")
				continue
			}
			latest_status[tweet.User().ScreenName()] = tweet
		}
		max_id = max_id - 1
		fmt.Fprintf(w, "Got %v Tweets", batch)
		if resp.HasRateLimit() {
			fmt.Fprintf(w, ", %v calls available", resp.RateLimitRemaining())
		}
		fmt.Fprintf(w, ".\n")
	}

	for _, tweet := range latest_status {
		opts := datastore.TransactionOptions{XG: true}
		if err := datastore.RunInTransaction(c, func(c appengine.Context) error {
			driverKey := datastore.NewKey(c, "Driver", tweet.User().ScreenName(), 0, nil)
			driver := new(Driver)
			if err := datastore.Get(c, driverKey, driver); err != nil {
				return err
			}
			car := new(Car)
			if err := datastore.Get(c, driver.CurrentCar, car); err != nil {
				return err
			}
			car.Action = strings.ToLower(strings.Fields(tweet.Text()[len("@"+bot.Name):])[0])
			car.Updated = tweet.CreatedAt()
			if _, err := datastore.Put(c, driver.CurrentCar, car); err != nil {
				return err
			}
			fmt.Fprintf(w, "%s (%d): %s\n", car.Driver, driver.CurrentCar.Parent().IntID(), car.Action)
			commands[driver.CurrentCar.Parent().IntID()] = struct{}{}
			return err
		}, &opts); err != nil {
			fmt.Fprintf(w, "%v\n", err)
			continue
		}
	}

	for command, _ := range commands {
		updateCommand(w, c, client, command, updated)
	}
}

func updateCommandHandler(w http.ResponseWriter, r *http.Request) {
	var (
		commandId int64
		client    *twittergo.Client
		err       error
	)
	if id, err := strconv.Atoi(r.FormValue("id")); err == nil {
		commandId = int64(id)
	} else {
		return
	}
	c := appengine.NewContext(r)
	if client, _, err = LoadCredentials(c); err != nil {
		fmt.Fprintf(w, "Error loading credentials: %v\n", err)
		return
	}
	updatedTime := time.Now()
	updateCommand(w, c, client, commandId, updatedTime)
}

func updateCommand(w http.ResponseWriter, c appengine.Context, client *twittergo.Client, commandId int64, updated time.Time) {
	var (
		spotsParked int
		cars        []Car
	)
	fmt.Fprintf(w, "Updating %d\n", commandId)

	commandKey := datastore.NewKey(c, "Command", "", commandId, nil)
	command := new(Command)
	if err := datastore.Get(c, commandKey, command); err != nil {
		fmt.Fprintf(w, "%v\n", err)
		return
	}

	if err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		q := datastore.NewQuery("Car").Ancestor(commandKey).Order("Updated")
		carKeys, err := q.GetAll(c, &cars)
		if err != nil {
			fmt.Fprintf(w, "%v\n", err)
			return err
		}
		fmt.Fprintf(w, "Cars before update: \n%v\n", cars)
		// Find number of parked spots
		for i := range cars {
			if cars[i].Parked != "" && cars[i].Parked != "street" {
				spotsParked++
			}
		}
		// Update parking using actions
		for i := range cars {
			if cars[i].Updated.Before(command.Updated) {
				continue
			}
			if cars[i].Action == "park" {
				if cars[i].Parked != "" && cars[i].Parked != "street" {
					continue
				}
				cars[i].Parked = fmt.Sprintf("%d", spotsParked)
				spotsParked++
			} else if cars[i].Action == "street" {
				cars[i].Parked = "s"
			} else if cars[i].Action == "gone" {
				if cars[i].Parked != "" && cars[i].Parked != "street" {
					cars[i].Parked = ""
					spotsParked--
				}
			}
		}
		if _, err := datastore.PutMulti(c, carKeys, cars); err != nil {
			return err
		}
		fmt.Fprintf(w, "Cars after update: \n%v\n", cars)
		return err
	}, nil); err != nil {
		fmt.Fprintf(w, "%v\n", err)
		return
	}
	// Update command
	command.Updated = updated
	if _, err := datastore.Put(c, commandKey, command); err != nil {
		fmt.Fprintf(w, "%v\n", err)
		return
	}
	// Get suggestions
	updates := suggestSpots(cars, command.Spots)
	fmt.Fprintf(w, "Updates: \n%v\n", updates)
}

func suggestSpots(cars []Car, numSpots int) (results []string) {
	sort.Sort(BySchedule(cars))
	suggestions := make(map[string]string)
	targetCars := make([]Car, 0, len(cars))
	carParked := false
	spotsParked := 0
	for i := range cars {
		if cars[i].Parked == "" {
			if !carParked {
				targetCars = append(targetCars, cars[i])
			} else {
				suggestions[cars[i].Driver] = "s"
			}
		} else if cars[i].Parked != "street" {
			carParked = true
			spotsParked++
		}
	}
	spotsAvailable := numSpots - spotsParked
	if spotsAvailable <= 1 {
		var suggestion string
		if spotsAvailable == 0 {
			suggestion = "s"
		} else {
			if len(targetCars) == 1 {
				suggestion = "i"
			} else {
				suggestion = "s/i"
			}
		}
		for i := range targetCars {
			suggestions[targetCars[i].Driver] = suggestion
		}
	} else {
		carsBefore := 0
		carsAfter := len(targetCars) - 1
		for i := range targetCars {
			if carsBefore-carsAfter < 0 {
				suggestions[targetCars[i].Driver] = "s"
			} else {
				suggestions[targetCars[i].Driver] = "i"
			}
			carsBefore++
			carsAfter--
		}
	}
	for driver, suggestion := range suggestions {
		switch {
		case suggestion == "s":
			results = append(results, fmt.Sprintf("@%s Please street park", driver))
		case suggestion == "i":
			results = append(results, fmt.Sprintf("@%s Please park inside", driver))
		case suggestion == "s/i":
			results = append(results, fmt.Sprintf("@%s Please street park if possible", driver))
		}
	}
	return
}
