// TODO: Separate routes out into other fuctions and so on.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var (
	db     *sql.DB
	config = struct {
		StaticRoot string `json:"staticRoot"`
		Cert       string `json:"cert"`
		Key        string `json:"key"`
		DbUserName string `json:"dbUserName`
		DbPass     string `json:"dbPass"`
	}{}
)

func init() {
	var err error
	die := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	// Load configuration from js.
	data, err := ioutil.ReadFile("config.json")
	die(err)
	err = json.Unmarshal(data, &config)
	fmt.Println(config)
	die(err)
	connString := fmt.Sprint("user=", config.DbUserName, " password=", config.DbPass, " dbname=eaglelist")
	db, err = sql.Open("postgres", connString)
	die(err)
	die(db.Ping())
}

type User struct {
	Handle   string
	Email    string
	Bio      string
	ImageURL string
}

type Listing struct {
	User
	ListingID  int
	UserID     int
	Title      string
	Content    string
	CreateDate time.Time
	EndDate    time.Time
}

func main() {
	// Set up the routes
	router := httprouter.New()
	fmt.Println(config)
	fmt.Println("Starting server")
	router.ServeFiles("/static/*filepath", http.Dir(config.StaticRoot))
	router.HandlerFunc("GET", "/apidb/users", loadDataFromDB)
	router.GET("/apidb/users/id/:id", userByID)
	router.GET("/apidb/users/handle/:user", searchUsers)
	router.GET("/apidb/listings", allListings)
	router.GET("/apidb/listings/:id/id", listingsByID)
	router.GET("/", index)

	log.Fatal(http.ListenAndServeTLS(":443", config.Cert, config.Key, router))
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, `API Routes:
  GET "/" -> This file.
  GET "/static/*filepath" -> Old hard-coded api, other static assets
  GET "/apidb/users" -> returns a list of all users
  GET "/apidb/users/id/:id" -> returns a list of all users
  GET "/apidb/users/handle/:user" -> searched for users that have handle's matchin said patter.
  GET "/apidb/listings" -> Returns a list of all listings 
	`)
}

func allListings(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	rows, err := db.Query(`
	Select 
		listings.id as listingID,
		handle,
		email,
		imageURL,
		title,
		content, 
		users.id as userID,
		createdate,
		enddate
	from listings
	inner join users on users.id = listings.creatorID
	where listings.newversionid = -1;
	`)
	if err != nil {
		fmt.Print(err)
		w.WriteHeader(500)
		return
	}
	emitListings(w, rows)
}

func listingsByID(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id, err := strconv.Atoi(p[0].Value)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	rows, err := db.Query(`
	Select 
		listings.id as listingID,
		handle,
		email,
		imageURL,
		title,
		content, 
		users.id as userID,
		createdate,
		enddate
	from listings
	inner join users on users.id = listings.creatorID
	where listings.newversionid = -1 and listings.id = $1;
	`, id)
	if err != nil {
		fmt.Print(err)
		w.WriteHeader(500)
		return
	}
	emitListings(w, rows)
}

func emitListings(w http.ResponseWriter, rows *sql.Rows) {
	listings := make([]Listing, 0)
	for rows.Next() {
		listing := Listing{}
		rows.Scan(
			&listing.ListingID,
			&listing.Handle,
			&listing.Email,
			&listing.ImageURL,
			&listing.Title,
			&listing.Content,
			&listing.UserID,
			&listing.CreateDate,
			&listing.EndDate)
		listings = append(listings, listing)
	}
	data, err := json.Marshal(struct{ Listings []Listing }{listings})
	if err != nil {
		fmt.Print(err)
		w.WriteHeader(500)
		return
	}
	w.Write(data)
}

func emitUsers(w http.ResponseWriter, rows *sql.Rows) {
	users := make([]User, 0)
	for rows.Next() {
		user := User{}
		rows.Scan(&user.Handle, &user.Email, &user.Bio, &user.ImageURL)
		users = append(users, user)
	}
	rows.Close()
	data, err := json.Marshal(struct{ Users []User }{users})
	// TODO: clean this up.
	if err != nil {
		panic(err)
	}
	w.Write(data)
}

func userByID(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id, err := strconv.Atoi(p[0].Value)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	fmt.Println(id)

	rows, err := db.Query(`Select handle, email, biography, imageURL
	from users,
	where id = $1`)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	emitUsers(w, rows)
}

// Search through the users table by handle
func searchUsers(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	userName := strings.Replace(p[0].Value, "*", "%", -1)
	fmt.Println(userName)
	rows, err := db.Query(`Select handle, email, biography, imageURL 
	from users
	where handle like $1
	`, userName)
	if err != nil {
		panic(err)
	}
	emitUsers(w, rows)
}

// Pull all the user data from the database.
func loadDataFromDB(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("Select handle, email, biography, imageURL from users")
	if err != nil {
		panic(err)
	}
	emitUsers(w, rows)
}
