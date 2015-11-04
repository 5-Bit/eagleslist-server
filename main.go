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
	router.POST("/apidb/users/new", newUser)
	router.PUT("/apidb/users/auth", authUser)
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

  PUT "/apidb/users/auth" -> PUT a JSON object with "UserHandle" and "Password" fields, will return an object that has an "Error" key, and optionally an "UserID" and "SessionKey" keys
  POST "/apidb/users/new" -> create a new users with the specified information. Returns the users's ID and a session cookie
	`)
}

// Pull all the user data from the database.
func loadDataFromDB(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("Select handle, email, biography, imageURL from users")
	if err != nil {
		panic(err)
	}
	emitUsers(w, rows)
}
