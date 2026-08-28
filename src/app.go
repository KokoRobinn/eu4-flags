package main

import (
	"database/sql"
	"log"
	"math/rand/v2"
	"net/http"
	"text/template"

	_ "github.com/mattn/go-sqlite3"
)

type Question struct {
	Flag   string
	Answer string
	Idx    int
}

const DB_NAME string = "countries.db"

var tags []string

func get_tags(db *sql.DB) {
	count, err := db.Query("SELECT COUNT(*) FROM Countries")
	if err != nil {
		log.Fatal(err)
	}

	var num_countries int = 0
	count.Next()
	if err := count.Scan(&num_countries); err != nil {
		log.Fatal(err)
	}

	tags = make([]string, num_countries)

	rows, err := db.Query("SELECT tag FROM Countries")
	if err != nil {
		log.Fatal(err)
	}

	var i int = 0
	for rows.Next() {
		if err := rows.Scan(&tags[i]); err != nil {
			log.Fatal(err)
		}
		i++
	}
}

func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

// Creates a random question, barring country indices present in recently_guessed
func random_question(db *sql.DB, recently_guessed []int) Question {
	var r = rand.IntN(len(tags))
	for Contains(recently_guessed, r) {
		r = r + 1%len(tags)
	}
	var new_tag string = tags[r]

	rows, err := db.Query("SELECT name, flag_path FROM Countries WHERE tag=?", new_tag)
	if err != nil {
		log.Fatal(err)
	}

	var name string
	var flag_path string
	rows.Next()
	if err := rows.Scan(&name, &flag_path); err != nil {
		log.Fatal(err)
	}

	return Question{flag_path, name, r}
}

func main() {
	db, err := sql.Open("sqlite3", "./"+DB_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	main_tmpl := template.Must(template.ParseFiles("main.html"))

	get_tags(db)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			var q = random_question(db, make([]int, 0))
			main_tmpl.Execute(w, struct {
				Question Question
			}{q})
		}
	})

	fs := http.FileServer(http.Dir("/static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":8787", nil)
}
