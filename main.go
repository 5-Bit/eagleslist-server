// TODO: Separate routes out into other fuctions and so on.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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
	die(err)
	connString := fmt.Sprint("user=", config.DbUserName, " password=", config.DbPass, " dbname=eaglelist")
	db, err = sql.Open("postgres", connString)
	die(err)
	die(db.Ping())
}

func main() {
	// Set up the routes
	router := httprouter.New()
	fmt.Println("Starting server")
	router.ServeFiles("/static/*filepath", http.Dir(config.StaticRoot))
	// router.HandlerFunc("GET", "/apidb/users", loadDataFromDB)
	router.GET("/verify/:verifyToken", verifyUser)
	router.GET("/apidb/listings", allListings)
	router.GET("/apidb/listings/:id/id", listingsByID)
	router.POST("/apidb/listings/new", newListing)
	router.GET("/apidb/searchlistings/:search", searchListings)
	// Direct message routes
	router.POST("/apidb/listing/:od/adddirectmessage", directMessageToListing)
	router.PUT("/apidb/listing/:ud/getdirectmessages", getDirectMessagesForListing)

	// Routes for listing comments
	router.POST("/apidb/listingcomments/:id/add", commentToListing)
	router.GET("/apidb/listingcomments/:id/getAll", getCommentsForListing)
	router.PUT("/apidb/deletecomment/:id/", deleteCommentOnListing)

	// Routes for users.
	router.GET("/apidb/users/handle/:user", searchUsers)
	router.PUT("/apidb/users/id/:id", userByID)
	router.PUT("/apidb/users/directmessages/", getDirectMessagesForListing)
	router.PUT("/apidb/users/auth", authUser)
	router.PUT("/apidb/users/logout", invalidateSession)
	router.POST("/apidb/users/new", newUser)
	//router.PUT("/apidb/validation/resend/", resendAPI)
	router.GET("/", index)
	loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, router)

	log.Fatal(http.ListenAndServeTLS(":443", config.Cert, config.Key, loggedRouter))
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, `API Routes:
  GET "/" -> This file.

  GET "/static/*filepath" -> Old hard-coded api, other static assets

  *Removed* GET "/apidb/users" -> returns a list of all users

  PUT "/apidb/users/id/:id" -> returns the user that has that ID, or an error.
  GET "/apidb/users/handle/:user" -> searched for users that have handle's matchin said patter.
  PUT "/apidb/users/auth" -> PUT a JSON object with "UserHandle" and "Password" fields, will return an object that has an "Error" key, and optionally an "UserID" and "SessionID" keys
  PUT "/apidb/users/logout" -> PUT a JSON object with the session key to invalidate that session key.


  PUT "/apidb/users/directmessages" -> PUT a JSON object with the session key to get a list of all direct messages for that user.
  POST "/apidb/users/new" -> Create a new users with the specified information.  Returns the users's ID and a session cookie


  GET "/apidb/searchlistings/:search" -> GET, returns an error code and a list of listings that match this search.
  POST "/apidb/listing/:id/adddirectmessage" -> POST JSON containing a SessionID and DirectMessage object to add to the listingID. Returns an object containing an error, if any happened.
  PUT "/apidb/listing/:id/getdirectmessage" -> PUT JSON containing a SessionID and DirectMessage object to add to the listingID. Returns an object containing an error, if any happened.


  GET "/apidb/listings" -> Returns a list of all listings 
  GET "/apidb/listings/:id/id" -> Returns a list of a single listing, based on the id in the URL.
  POST "/apidb/listingcomments/:id/add" -> POST JSON containing a SessionID and a Comment object that contains a Content field, returns 
  PUT "/apidb/listingcomments/:id/getAll" -> POST JSON containing a SessionID. Returns an object that has Error and Comments keys.
  *new* PUT "/apidb/deletecomment/:id/" -> PUT a JSON contiaining a SessionID. 


  POST "/apidb/listings/new" -> Create a listing from the JSON passed to it, return the id for the listing.
  
	`)
}
