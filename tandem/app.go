package tandem

import (
	"html/template"
	"net/http"
)

func init() {
	http.HandleFunc("/", index)
	http.HandleFunc("/command/", command)
}

type CommandPage struct {
	Id string
}

func index(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, nil)
}

func command(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/command/"):]
	p := &CommandPage{Id: id}
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, p)
}
