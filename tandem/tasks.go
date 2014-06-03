package tandem

import (
	"fmt"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"appengine"
	"appengine/datastore"
)

func init() {
	http.HandleFunc("/tasks/update", update)
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
			if i == 0 {
				bot.LastUpdateId = tweet.IdStr()
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
}
