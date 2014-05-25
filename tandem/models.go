package tandem

import (
	"time"
)

type Command struct {
	Name  string `json:"name"`
	Spots int    `json:"spots"`
}

type Car struct {
	Driver   string    `json:"driver"`
	Schedule time.Time `json:"schedule"`
	Parked   string    `json:"parked"`
}
