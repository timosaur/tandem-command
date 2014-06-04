package tandem

import (
	"time"

	"appengine/datastore"
)

type Bot struct {
	BotKey         string
	BotSecret      string
	ConsumerKey    string
	ConsumerSecret string
	LastUpdateId   string
	Name           string
}

type Command struct {
	Name    string `json:"name"`
	Spots   int    `json:"spots"`
	Updated time.Time
}

type Car struct {
	DriverKey *datastore.Key
	Driver    string    `json:"driver"`
	Schedule  time.Time `json:"schedule"`
	Parked    string    `json:"parked"`
	Action    string
	Updated   time.Time
}

type Driver struct {
	CurrentCar *datastore.Key
}
