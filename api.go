package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"strings"

	// FIXME: This import path will currently only work on this server.
	"yumaikas/eaglelist/eagleslist-server/templates"
	email "yumaikas/eaglelist/eagleslist-server/validation"

	_ "github.com/lib/pq"
)

// Unmarshal JSON
func decodeJSON(w http.ResponseWriter, r *http.Request, o interface{}) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	err := json.Unmarshal(buf.Bytes(), o)
	if err != nil {
		fmt.Println(string(buf.Bytes()))
		fmt.Println(err)
		writeJsonERR(w, 400, "Invalid JSON!")
		return err
	}
	return nil
}

// Write an error as a JSON object
func writeJsonERR(w http.ResponseWriter, Header int, message string) {
	w.WriteHeader(Header)
	errObj := &struct {
		Error string
	}{
		message,
	}
	data, err := json.Marshal(errObj)
	if err != nil {
		panic(err)
	}
	w.Write(data)
}

func newUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	buf := new(bytes.Buffer)
	newUser := &struct {
		UserName string
		Email    string
		Password string
	}{}
	buf.ReadFrom(r.Body)
	err := json.Unmarshal(buf.Bytes(), newUser)
	if err != nil {
		fmt.Println(string(buf.Bytes()))
		fmt.Println(err)
		writeJsonERR(w, 400, "Invalid JSON!")
		return
	}

	if !(strings.HasSuffix(newUser.Email, "@fgcu.edu") ||
		strings.HasSuffix(newUser.Email, "@eagle.fgcu.edu")) {
		writeJsonERR(w, 400, "Email needs to be from FGCU!")
		return
	}

	passData, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), 13)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	var count int
	err = db.QueryRow(`Select count(id) from users where email = $1 `, newUser.Email).Scan(&count)
	if err != nil || count > 0 {
		fmt.Println(err)
		writeJsonERR(w, 400, "User Exists!")
		return
	}

	var userID int
	err = db.QueryRow(`
	Insert into users (handle, email, passwordbcrypt) VALUES($1, $2, $3) RETURNING ID
	`, newUser.UserName, newUser.Email, passData).Scan(&userID)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Unable to create user!")
		return
	}
	// Prepare for the email validation.
	validation, err1 := GenerateRandomString()
	_, err = db.Exec(`
	 Insert into emailvalidation (userid, validationtoken, isvalidated) VALUES ($1, $2, false);
	`, userID, validation)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Unknown error!")
		return
	}

	// Populate the confirmation email.
	emailBody, err := templates.GetConfirmationEmail(newUser.UserName, newUser.Email, validation)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Uknown(sic) error!")
		return
	}
	// Queue up the message send to be processed.
	email.SendMessage(emailBody, newUser.UserName)

	sessionKey, err1 := GetSessionKey(userID)
	if err1 != nil {
		sessionKey = "INVALID"
	}

	userInfo := &struct {
		Error   string
		UserId  int
		Session string
	}{"", userID, sessionKey}
	data, err := json.Marshal(userInfo)
	if err != nil {
		panic(err)
	}
	w.Write(data)
}

// Autheticate User
func authUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	buf := new(bytes.Buffer)
	auth := &struct {
		UserHandle string
		Password   string
	}{}
	buf.ReadFrom(r.Body)
	err := json.Unmarshal(buf.Bytes(), auth)
	if err != nil {
		writeJsonERR(w, 400, "Invalid JSON!")
		return
	}
	var passBuf []byte
	var id int
	err = db.QueryRow(`Select passwordbcrypt, id from users where email = $1 or handle = $1`, auth.UserHandle).Scan(&passBuf, &id)
	if err == sql.ErrNoRows {
		err = bcrypt.CompareHashAndPassword(passBuf, []byte(auth.Password))
		writeJsonERR(w, 400, "User name or password not found")
		return
	}
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Server error!")
		return
	}
	err = bcrypt.CompareHashAndPassword(passBuf, []byte(auth.Password))
	if err != nil {
		writeJsonERR(w, 400, "User name or password not found")
		return
	}

	sessionKey, err1 := GetSessionKey(id)
	userAuth := &struct {
		Error     string
		UserID    int
		SessionID string
	}{"", id, sessionKey}

	if err1 != nil {
		panic(err1)
	}

	data, err := json.Marshal(userAuth)
	if err != nil {
		panic(err)
	}
	w.Write(data)
}

// TODO: When creating login requests,
// make sure to have a special response for unvalidated users.
// func tryLogin

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

type listingResponse struct {
	User
	Listing
}

func emitListings(w http.ResponseWriter, rows *sql.Rows) {
	listings := make([]listingResponse, 0)
	for rows.Next() {
		listing := listingResponse{}
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
	data, err := json.Marshal(struct{ Listings []listingResponse }{listings})
	if err != nil {
		fmt.Print(err)
		w.WriteHeader(500)
		return
	}
	w.Write(data)
}

type EMIT_TYPE int

const (
	EMIT_MANY EMIT_TYPE = 1
	EMIT_ONE  EMIT_TYPE = 0
)

func emitUsers(w http.ResponseWriter, rows *sql.Rows, em EMIT_TYPE) {
	users := make([]User, 0)
	for rows.Next() {
		user := User{}
		rows.Scan(&user.Handle, &user.Email, &user.Bio, &user.ImageURL)
		users = append(users, user)
	}
	rows.Close()
	var data []byte
	var err error
	if em == EMIT_MANY {
		data, err = json.Marshal(struct{ Users []User }{users})
	} else {
		data, err = json.Marshal(users[0])
	}
	// TODO: clean this up.
	if err != nil {
		writeJsonERR(w, 500, "Ukown error")
	}
	w.Write(data)
}

// Create a new listing.
func newListing(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	req := &struct {
		SessionID string
		Listing   Listing
	}{}

	if err := decodeJSON(w, r, req); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}

	if ok, err := CheckSessionsKey(req.SessionID); !ok || err != nil {
		writeJsonERR(w, 400, "Unauthorized")
		return
	}

	var listingID int
	err := db.QueryRow(`
	Insert into listings (
		oldversionid, newversionid, creatorid, content,
		createdate, title, price
	) values (
		-1,         -- oldversionid
		-1,         -- newversionid
		$1,         -- creatorid
		$2,         -- content
		DATE 'NOW', -- createdate
		$3,         -- title
		$4          -- price
	)
	RETURNING ID
	`, req.Listing.UserID,
		req.Listing.Content,
		req.Listing.Title,
		req.Listing.Price).Scan(&listingID)

	if err != nil {
		writeJsonERR(w, 500, "UNKNOWN ERROR!")
		return
	}
	retVal := &struct {
		Error  string
		PostId int
	}{"", listingID}
	data, err := json.Marshal(retVal)
	if err != nil {
		writeJsonERR(w, 500, "UKNOWN error...")
		return
	}
	w.Write(data)
}

func userByID(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	auth := &struct {
		SessionID string
	}{}

	if err := decodeJSON(w, r, auth); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}

	// Check that authentication is working.
	if ok, err := CheckSessionsKey(auth.SessionID); !ok || err != nil {
		fmt.Println(err)
		writeJsonERR(w, 400, "Invalid auth token")
		return
	}

	id, err := strconv.Atoi(p[0].Value)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Invalid user ID")
		return
	}
	fmt.Println(id)

	rows, err := db.Query(`Select handle, email, biography, imageURL
	from users
	where id = $1`, id)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Unable to load user from database!")
		return
	}
	emitUsers(w, rows, EMIT_ONE)
}

// Search through the users table by handle
func searchUsers(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	auth := &struct {
		SessionID string
	}{}

	if err := decodeJSON(w, r, auth); err != nil {
		return
	}

	// Check to see if this user has
	if ok, err := CheckSessionsKey(auth.SessionID); !ok || err != nil {
		fmt.Println(err)
		writeJsonERR(w, 400, "Invalid auth token")
		return
	}

	userName := strings.Replace(p[0].Value, "*", "%", -1)
	fmt.Println(userName)
	rows, err := db.Query(`Select handle, email, biography, imageURL 
	from users
	where handle like $1
	`, userName)
	if err != nil {
		panic(err)
	}
	emitUsers(w, rows, EMIT_MANY)
}
