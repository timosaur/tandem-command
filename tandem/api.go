package tandem

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"appengine"
	"appengine/datastore"
)

func init() {
	http.HandleFunc("/save", save)
	http.HandleFunc("/cars", cars)
	http.HandleFunc("/driver", driver)
}

func save(w http.ResponseWriter, r *http.Request) {
	key, err := strconv.Atoi(r.FormValue("id"))
	if err != nil && r.Method != "POST" {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c := appengine.NewContext(r)
	k := datastore.NewKey(c, "Command", "", int64(key), nil)
	command := new(Command)
	if r.Method == "GET" {
		if err := datastore.Get(c, k, command); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "PUT" || r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&command); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		k, err = datastore.Put(c, k, command)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonresp, err := json.Marshal(command)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/json; charset=utf-8")
	w.Header().Set("Location", fmt.Sprintf("%s/save?id=%d", r.Host, k.IntID()))
	fmt.Fprintf(w, "%s", jsonresp)
}

func cars(w http.ResponseWriter, r *http.Request) {
	key, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c := appengine.NewContext(r)
	commandKey := datastore.NewKey(c, "Command", "", int64(key), nil)

	var cars []Car
	if r.Method == "GET" {
		q := datastore.NewQuery("Car").Ancestor(commandKey)
		if _, err := q.GetAll(c, &cars); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if r.Method == "PUT" || r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&cars); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, car := range cars {
			driverKey := datastore.NewKey(c, "Driver", car.Driver, 0, nil)
			q := datastore.NewQuery("Car").Ancestor(commandKey).
				Filter("DriverKey =", driverKey).
				KeysOnly()
			t := q.Run(c)
			carKey, _ := t.Next(nil)
			if carKey == nil {
				carKey = datastore.NewIncompleteKey(c, "Car", commandKey)
			}
			car.DriverKey = driverKey
			if _, err := datastore.Put(c, carKey, &car); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jsonresp, err := json.Marshal(cars)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/json; charset=utf-8")
	fmt.Fprintf(w, "%s", jsonresp)
}

func driver(w http.ResponseWriter, r *http.Request) {
	driverName := r.FormValue("name")
	currentCarId, err := strconv.Atoi(r.FormValue("car"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c := appengine.NewContext(r)
	driverKey := datastore.NewKey(c, "Driver", driverName, 0, nil)
	driver := new(Driver)
	datastore.Get(c, driverKey, driver)
	carKey := datastore.NewKey(c, "Car", "", int64(currentCarId), nil)
	driver.CurrentCar = carKey
	datastore.Put(c, driverKey, driver)
}
