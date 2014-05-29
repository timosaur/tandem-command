package tandem

import (
	"time"

	"appengine/datastore"
)

type Command struct {
	Name  string `json:"name"`
	Spots int    `json:"spots"`
}

type Car struct {
	CommandKey *datastore.Key
	DriverKey  *datastore.Key
	Driver     string    `json:"driver"`
	Schedule   time.Time `json:"schedule"`
	Parked     string    `json:"parked"`
}
