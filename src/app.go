package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"text/template"

	_ "github.com/mattn/go-sqlite3"
)

type Question struct {
	Flag   string `json:"flag"`
	Answer string `json:"answer"`
	Idx    int    `json:"idx"`
}

type Filter interface {
	appendSQL(q string, args []any) (string, []any)
}

type BoolFilter struct {
	Field   string
	Include bool
}

type ListFilter struct {
	Field   string
	Include []string
}

func (f BoolFilter) appendSQL(q string, args []any) (string, []any) {
	var val int = 0
	if f.Include {
		val = 1
	}
	return q + f.Field + "=?", append(args, val)
}

func (f ListFilter) appendSQL(q string, args []any) (string, []any) {
	q += f.Field + " IN ("
	for _, val := range f.Include {
		q += "?,"
		args = append(args, val)
	}
	q = q[:len(q)-1] + ")"
	return q, args
}

const DB_PATH string = "/db/countries.db"

var all_ids []int

func get_ids[T Filter](db *sql.DB, filters []T) []int {
	var where_query string = ""
	var query_args []any = make([]any, 0)
	if len(filters) > 0 {
		where_query = " WHERE "
	}
	for i, f := range filters {
		where_query, query_args = f.appendSQL(where_query, query_args)
		if i < len(filters)-1 {
			where_query += " AND "
		}
	}
	fmt.Fprintln(os.Stdout, where_query)
	fmt.Fprintln(os.Stdout, query_args)

	count, err := db.Query("SELECT COUNT(*) FROM Countries"+where_query, query_args...)
	if err != nil {
		log.Fatal(err)
	}

	var num_countries int = 0
	count.Next()
	if err := count.Scan(&num_countries); err != nil {
		log.Fatal(err)
	}

	var ids = make([]int, num_countries)

	rows, err := db.Query("SELECT id FROM Countries"+where_query, query_args...)
	if err != nil {
		log.Fatal(err)
	}

	var i int = 0
	for rows.Next() {
		if err := rows.Scan(&ids[i]); err != nil {
			log.Fatal(err)
		}
		i++
	}
	return ids
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
func random_question(db *sql.DB, ids []int, recently_guessed []int) Question {
	var r = rand.IntN(len(ids))
	for Contains(recently_guessed, r) {
		r = r + 1%len(ids)
	}
	var id int = ids[r]

	rows, err := db.Query("SELECT name, flag_path FROM Countries WHERE id=?", id)
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
	//var bf = BoolFilter{"formable", false}
	//var lf = ListFilter{"capital_subcontinent", []string{"Western Europe", "Eastern Europe"}}
	//var filters []Filter = []Filter{bf, lf}
	var filters []Filter = make([]Filter, 0)

	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	main_tmpl := template.Must(template.ParseFiles("main.html"))

	all_ids = get_ids(db, filters)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			main_tmpl.Execute(w, struct{}{})
		}
	})

	http.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			var q = random_question(db, all_ids, make([]int, 0))
			json.NewEncoder(w).Encode(q)
		}
	})

	fs := http.FileServer(http.Dir("/static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":8787", nil)
}
