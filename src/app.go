package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"text/template"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Question struct {
	Flag   string `json:"flag"`
	Answer string `json:"answer"`
	Idx    int    `json:"idx"`
}

type IdsEntry struct {
	Ids     []int
	Created time.Time
}

type Filter interface {
	appendSQL(q string, args []any) (string, []any)
}

type BoolFilter struct {
	Field   string `json:"field"`
	Include bool   `json:"include"`
}

type ListFilter struct {
	Field   string   `json:"field"`
	Include []string `json:"include"`
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

type Filters []Filter

func (f *Filters) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for _, item := range raw {
		var boolFilter BoolFilter
		if err := json.Unmarshal(item, &boolFilter); err == nil {
			var include json.RawMessage

			var obj map[string]json.RawMessage
			if err := json.Unmarshal(item, &obj); err != nil {
				return err
			}

			include = obj["include"]

			var b bool
			if err := json.Unmarshal(include, &b); err == nil {
				boolFilter.Include = b
				*f = append(*f, boolFilter)
				continue
			}
		}

		// Otherwise try ListFilter
		var listFilter ListFilter
		if err := json.Unmarshal(item, &listFilter); err != nil {
			return err
		}

		*f = append(*f, listFilter)
	}

	return nil
}

const DB_PATH string = "/db/countries.db"
const PLAYER_HASH_LEN = 10
const PLAYER_HASH_CHARS = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var all_filters []Filter = []Filter{
	ListFilter{"capital_subcontinent", []string{
		"-",
		"Western Europe",
		"Eastern Europe",
		"Levant",
		"India",
		"Amazonia",
		"North America",
		"Andes",
		"Central America",
		"Oceania",
		"Persia",
		"Tartary",
		"Northern Africa",
		"Southern Africa",
		"East Indies",
		"Far East",
		"China",
	}},
	BoolFilter{"formable", true},
	BoolFilter{"exists_1444", true},
	BoolFilter{"releasable", true},
}
var player_filters map[string]IdsEntry = make(map[string]IdsEntry, 0)
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

func make_player_hash(name string) string {
	seed := time.Now().Local().UnixMicro()
	player_hash := make([]byte, PLAYER_HASH_LEN)
	player_hash[0] = PLAYER_HASH_CHARS[seed%int64(len(PLAYER_HASH_CHARS))]
	for i := int64(1); i < PLAYER_HASH_LEN; i++ {
		player_hash[i] = PLAYER_HASH_CHARS[(int64(player_hash[i-1])*seed^int64(name[i%int64(len(name))])>>i)%int64(len(PLAYER_HASH_CHARS))]
	}
	//TODO: make this more robust
	//for _, exists := active_quizzes[Code(string(player_hash))]; exists; {
	//      player_hash[0] = PLAYER_HASH_CHARS[(int64(player_hash[0])^seed)%int64(len(CODE_CHARS))]
	//}
	return string(player_hash)
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

func get_player_hash(w http.ResponseWriter, r *http.Request) string {
	var player_hash_cookie, err = r.Cookie("player_hash")
	fmt.Fprintln(os.Stdout, "New connection from player")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			var player_hash = make_player_hash(r.RemoteAddr)
			player_hash_cookie = &http.Cookie{
				Name:    "player_hash",
				Value:   player_hash,
				Path:    "/",
				Expires: time.Now().Add(time.Hour * 24 * 365 * 10),
			}
			http.SetCookie(w, player_hash_cookie)
		default:
			http.Error(w, "server error", http.StatusInternalServerError)
			log.Fatal(err)
		}
	}
	return player_hash_cookie.Value
}

func purge_ids() {
	for h, e := range player_filters {
		if e.Created.Add(time.Hour * 24).Before(time.Now()) {
			delete(player_filters, h)
		}
	}
	time.Sleep(time.Hour * 3)
}

func main() {
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		log.Fatal(err)
	}
	go purge_ids()
	defer db.Close()

	main_tmpl := template.Must(template.ParseFiles("main.html"))

	all_ids = get_ids(db, make([]Filter, 0))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			get_player_hash(w, r)

			var bool_filters = make([]Filter, 0)
			var list_filters = make([]Filter, 0)
			for _, f := range all_filters {
				switch f.(type) {
				case BoolFilter:
					bool_filters = append(bool_filters, f)
				default:
					list_filters = append(list_filters, f)
				}
			}
			main_tmpl.Execute(w, struct {
				ListFilters []Filter
				BoolFilters []Filter
			}{list_filters, bool_filters})
		}
	})

	http.HandleFunc("/new", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			player_hash := get_player_hash(w, r)
			ids_entry, exists := player_filters[player_hash]
			var ids []int
			if !exists {
				ids = all_ids
			} else {
				ids = ids_entry.Ids
			}
			var q = random_question(db, ids, make([]int, 0))
			json.NewEncoder(w).Encode(q)
		}
	})

	http.HandleFunc("/update-settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			player_hash := get_player_hash(w, r)
			var filters Filters
			fmt.Fprintln(os.Stdout, "received new settings for "+player_hash)
			err := json.NewDecoder(r.Body).Decode(&filters)
			if err != nil {
				fmt.Fprintln(os.Stdout, "Invalid settings submitted")
				return
			}
			new_ids := get_ids(db, filters)
			player_filters[player_hash] = IdsEntry{new_ids, time.Now()}
		}
	})

	fs := http.FileServer(http.Dir("/static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":8787", nil)
}
