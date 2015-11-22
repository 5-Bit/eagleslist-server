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
	"github.com/5-bit/eagleslist-server/templates"
	email "github.com/5-bit/eagleslist-server/validation"

	_ "github.com/lib/pq"
)

/*
TODO: Figure out good ways to deduplicate the code below. There are a lot of variations of
very similar code.... :(
*/

// Unmarshal JSON
func decodeJSON(w http.ResponseWriter, r *http.Request, o interface{}) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	b := buf.Bytes()
	fmt.Println(string(b))
	err := json.Unmarshal(b, o)
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
	email.SendMessage(emailBody, newUser.Email)

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

// Authenticate User
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

func invalidateSession(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// TODO: finish this code
	auth := &struct {
		SessionID string
	}{}

	if err := decodeJSON(w, r, auth); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}
	_, err := db.Exec(`
	Update sessions
	set valid_til = TIMESTAMPTZ 'NOW' + INTERVAL '-1 MINUTE'
	where cookieinfo = $1`, auth.SessionID)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "ukerr")
		return
	}
	w.WriteHeader(204)
}

func verifyUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	verificationToken := p[0].Value
	var handle string
	var email string
	var userID int
	var isValidated bool
	err := db.QueryRow(`
	 Select userid, handle, email, isValidated
	 from emailvalidation 
	 inner join users on users.id = emailvalidation.userid 
	 	and validationtoken = $1;
	`, verificationToken).Scan(&userID, &handle, &email, &isValidated)
	if err == sql.ErrNoRows {
		w.WriteHeader(404)
		fmt.Fprint(w, "User not found")
		return
	}
	if isValidated {
		pageData, err := templates.GetLandingPage(handle, email)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprint(w, "unknown error........")
		}
		fmt.Fprint(w, pageData)
		return
	}
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		fmt.Fprint(w, "Unknown error")
		return
	}
	_, err = db.Exec(`
	Update emailvalidation
	set 
		isValidated = true
	where userid = $1;
	`, userID)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		fmt.Fprint(w, "Unknown error..")
		return
	}
	pageData, err := templates.GetVerificationPage(handle, email)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		fmt.Fprint(w, "Unknown error....")
		return
	}
	fmt.Fprint(w, pageData)
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
		price,
		condition,
		createdate,
		enddate
	from listings
	inner join users on users.id = listings.creatorID
	-- where listings.newversionid = -1
	order by createdate desc;
	`)
	if err != nil {
		fmt.Print(err)
		w.WriteHeader(500)
		return
	}
	emitListings(w, rows)
}

func searchListings(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	rows, err := db.Query(`
Select
	 listings.id as listingID,
	 handle,
	 email,
	 imageURL,
	 title,
	 content,
	 users.id as userID,
	 price,
	 condition,
	 createdate,
	 enddate
from listings
INNER JOIN users on users.id = listings.creatorid
where to_tsvector(title || ' ' ||  Content || ' ' || handle) @@ plainto_tsquery($1); `,
		p[0].Value)
	if err != nil {
		fmt.Print(err)
		writeJsonERR(w, 500, "Search failed!")
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
		price,
		condition,
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
		err := rows.Scan(
			&listing.ListingID,
			&listing.Handle,
			&listing.Email,
			&listing.ImageURL,
			&listing.Title,
			&listing.Content,
			&listing.UserID,
			&listing.Price,
			&listing.Condition,
			&listing.CreateDate,
			&listing.EndDate)
		if err != nil {
			fmt.Print(err)
			w.WriteHeader(500)
			return
		}
		listings = append(listings, listing)
	}
	fmt.Println(listings)
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

// Create a new book (This should be finished later).
func newBook(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	req := &struct {
		SessionID string
		Listing   Listing
	}{}

	if err := decodeJSON(w, r, req); err != nil {
		fmt.Println(err)
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}
	fmt.Println(req)

	if id, ok, err := CheckSessionsKey(req.SessionID); !ok || err != nil {
		fmt.Println(err)
		writeJsonERR(w, 400, "Unauthorized")
		return
	} else {
		req.Listing.UserID.Set(int64(id))
	}

	var listingID int
	err := db.QueryRow(`
	Insert into listings (
		oldversionid, newversionid, creatorid, content,
		createdate, title, price, condition
	) values (
		-1,         -- oldversionid
		-1,         -- newversionid
		$1,         -- creatorid
		$2,         -- content
		TIMESTAMPTZ 'NOW', -- createdate
		$3,         -- title
		$4,         -- price
		$5          -- condition
	) RETURNING ID;
	`, req.Listing.UserID,
		req.Listing.Content,
		req.Listing.Title,
		req.Listing.Price,
		req.Listing.Condition).Scan(&listingID)

	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "UNKNOWN ERROR!")
		return
	}
	retVal := &struct {
		Error  string
		PostID int
	}{"", listingID}
	data, err := json.Marshal(retVal)
	if err != nil {
		writeJsonERR(w, 500, "UKNOWN error...")
		return
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
		fmt.Println(err)
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}
	fmt.Println(req)

	if id, ok, err := CheckSessionsKey(req.SessionID); !ok || err != nil {
		fmt.Println(err)
		writeJsonERR(w, 400, "Unauthorized")
		return
	} else {
		req.Listing.UserID.Set(int64(id))
	}

	var listingID int
	err := db.QueryRow(`
	Insert into listings (
		oldversionid, newversionid, creatorid, content,
		createdate, title, price, condition
	) values (
		-1,         -- oldversionid
		-1,         -- newversionid
		$1,         -- creatorid
		$2,         -- content
		TIMESTAMPTZ 'NOW', -- createdate
		$3,         -- title
		$4,         -- price
		$5          -- condition
	) RETURNING ID;
	`, req.Listing.UserID,
		req.Listing.Content,
		req.Listing.Title,
		req.Listing.Price,
		req.Listing.Condition).Scan(&listingID)

	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "UNKNOWN ERROR!")
		return
	}
	retVal := &struct {
		Error  string
		PostID int
	}{"", listingID}
	data, err := json.Marshal(retVal)
	if err != nil {
		writeJsonERR(w, 500, "UKNOWN error...")
		return
	}
	w.Write(data)
}

// Delete a comment.
func deleteCommentOnListing(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	commentID, err := strconv.Atoi(p[0].Value)
	if err != nil {
		writeJsonERR(w, 400, "Invalid comment ID in URL")
		return
	}
	auth := &struct{ SessionID string }{}
	if err := decodeJSON(w, r, auth); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}

	var userID int
	if id, ok, err := CheckSessionsKey(auth.SessionID); !ok || err != nil {
		writeJsonERR(w, 400, "Unauthorized")
		return
	} else {
		userID = id
	}
	var creatorID int
	err = db.QueryRow(`
	Select creatorid from comments where id = $1;
	`, commentID).Scan(&creatorID)
	if err == sql.ErrNoRows {
		writeJsonERR(w, 400, "Listing does not exist")
		return
	} else if creatorID != userID {
		writeJsonERR(w, 400, "Not authorized!")
		return
	}
	if err != nil {
		writeJsonERR(w, 500, "UK ERR")
	}
	_, err = db.Exec(`Update comments set endDate = TIMESTAMPTZ 'NOW' where id = $1`)
	if err != nil {
		writeJsonERR(w, 500, "UNKER!")
		return
	}
}

// Add a comment to a listing.
func commentToListing(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	parentListingID, err := strconv.Atoi(p[0].Value)
	if err != nil {
		writeJsonERR(w, 400, "Invalid listing ID in URL")
		return
	}
	req := &struct {
		SessionID string
		Comment   Comment
	}{}

	if err := decodeJSON(w, r, req); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}

	if id, ok, err := CheckSessionsKey(req.SessionID); !ok || err != nil {
		writeJsonERR(w, 400, "Unauthorized")
		return
	} else {
		req.Comment.UserID.Set(int64(id))
	}

	err = db.QueryRow(`
	Select id from listings where id = $1;
	`, parentListingID).Scan(&parentListingID)
	if err == sql.ErrNoRows {
		writeJsonERR(w, 400, "Listing does not exist")
		return
	}
	if err != nil {
		writeJsonERR(w, 500, "UK ERR")
	}

	var listingID int
	err = db.QueryRow(`
	Insert into comments (
		oldversionid, newversionid, creatorid, content,
		createdate, parent_listing_id 
	) values (
		-1,         -- oldversionid
		-1,         -- newversionid
		$1,         -- creatorid
		$2,         -- content
		TIMESTAMPTZ 'NOW', -- createdate
		$3          -- parent_listing_id
	)
	RETURNING ID
	`, req.Comment.UserID,
		req.Comment.Content,
		parentListingID).Scan(&listingID)

	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "UNKNOWN ERROR!!")
		return
	}
	retVal := &struct {
		Error string
	}{""}
	data, err := json.Marshal(retVal)
	if err != nil {
		writeJsonERR(w, 500, "UKNOWN error...")
		return
	}
	w.Write(data)
}

type commentResponse struct {
	UserName NullString
	Comment
}

// /apidb/listings/:id/getdirectmessages
func getCommentsForListing(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	parentListingID, err := strconv.Atoi(p[0].Value)
	if err != nil {
		writeJsonERR(w, 400, "Invalid listing ID in URL")
		return
	}
	rows, err := db.Query(`
		Select
			comments.id, 
			creatorID,
			content,
			createDate,
			enddate,
			handle
		from comments
		inner join users on CreatorID = users.id
		where parent_listing_id = $1
		and enddate is not null
		`, parentListingID)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Unknown error!!!")
		return
	}

	comments := make([]commentResponse, 0)
	for rows.Next() {
		comment := commentResponse{}
		comment.ParentListingID.Set(int64(parentListingID))
		err = rows.Scan(
			&comment.ID,
			&comment.UserID,
			&comment.Content,
			&comment.CreateDate,
			&comment.EndDate,
			&comment.UserName)
		if err != nil {
			fmt.Println(err)
		}
		comments = append(comments, comment)
	}
	retVal := struct {
		Error    string
		Comments []commentResponse
	}{"", comments}
	data, err := json.Marshal(retVal)

	if err != nil {
		writeJsonERR(w, 500, "Json error!")
		return
	}
	w.Write(data)
}

// Attach a DM to a listing.
func directMessageToListing(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	parentListingID, err := strconv.Atoi(p[0].Value)
	if err != nil {
		writeJsonERR(w, 400, "Invalid listing ID in URL")
		return
	}
	req := &struct {
		SessionID     string
		DirectMessage DirectMessage
	}{}

	if err := decodeJSON(w, r, req); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}

	if id, ok, err := CheckSessionsKey(req.SessionID); !ok || err != nil {
		writeJsonERR(w, 400, "Unauthorized")
		return
	} else {
		req.DirectMessage.UserID.Set(int64(id))
	}

	err = db.QueryRow(`
	Select id from listings where id = $1;
	`, parentListingID).Scan(&parentListingID)
	if err == sql.ErrNoRows {
		writeJsonERR(w, 400, "Listing does not exist")
		return
	}
	if err != nil {
		writeJsonERR(w, 500, "UK ERR")
	}

	var listingID int
	err = db.QueryRow(`
	Insert into directmessages (
		oldversionid, newversionid, creatorid, content,
		createdate, parent_listing_id 
	) values (
		-1,         -- oldversionid
		-1,         -- newversionid
		$1,         -- creatorid
		$2,         -- content
		TIMESTAMPTZ 'NOW', -- createdate
		$3          -- parent_listing_id
	)
	RETURNING ID
	`, req.DirectMessage.UserID,
		req.DirectMessage.Content,
		parentListingID).Scan(&listingID)

	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "UNKNOWN ERROR!!")
		return
	}
	retVal := &struct {
		Error string
	}{""}
	data, err := json.Marshal(retVal)
	if err != nil {
		writeJsonERR(w, 500, "UKNOWN error...")
		return
	}
	w.Write(data)
}

// /apidb/listings/:id/getdirectmessages
func getDirectMessagesForListing(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	parentListingID, err := strconv.Atoi(p[0].Value)
	if err != nil {
		writeJsonERR(w, 400, "Invalid listing ID in URL")
		return
	}
	auth := &struct {
		SessionID string
	}{}

	if err := decodeJSON(w, r, auth); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}

	{
		// Check to make sure that the listing that all the direct messages are
		// being listed for belongs to the user in question.
		var userID int
		var ok bool
		var err error
		var listingOwnerid int
		err = db.QueryRow(` Select creatorid from listings where id = $1`,
			parentListingID).Scan(&listingOwnerid)
		// Check that authentication is working.
		if userID, ok, err = CheckSessionsKey(auth.SessionID); !ok || err != nil {
			fmt.Println(err)
			writeJsonERR(w, 400, "Invalid auth token")
			return
		}
		if userID != listingOwnerid {
			writeJsonERR(w, 500, "You can't look at the DMs for someone elses listing!")
			return
		}
	}
	rows, err := db.Query(`
		Select
			id, 
			creatorID,
			content,
			createDate,
			enddate
		from directmessages
		where parent_listing_id = $1
		`, parentListingID)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Unknown error!!!")
		return
	}

	directMessages := make([]DirectMessage, 0)
	for rows.Next() {
		dm := DirectMessage{}
		dm.ParentListingID.Set(int64(parentListingID))
		rows.Scan(
			&dm.ID,
			&dm.UserID,
			&dm.Content,
			&dm.CreateDate,
			&dm.EndDate)
		directMessages = append(directMessages, dm)
	}
	retVal := struct {
		Error          string
		DirectMessages []DirectMessage
	}{"", directMessages}
	data, err := json.Marshal(retVal)

	if err != nil {
		writeJsonERR(w, 500, "Json error!")
		return
	}
	w.Write(data)
}

// Get all the direct messages for a given user.
func getDirectMessagesForUser(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	auth := &struct {
		SessionID string
	}{}

	if err := decodeJSON(w, r, auth); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}

	// Check to make sure that the listing that all the direct messages are
	// being listed for belongs to the user in question.
	var userID int
	var ok bool
	var err error
	// Check that authentication is working.
	if userID, ok, err = CheckSessionsKey(auth.SessionID); !ok || err != nil {
		fmt.Println(err)
		writeJsonERR(w, 400, "Invalid auth token")
		return
	}
	rows, err := db.Query(`
		Select
			directmessages.id, 
			directmessages.creatorID,
			directmessages.content,
			directmessages.parent_listing_id,
			directmessages.createDate,
			directmessages.enddate
		from directmessages
		inner join listings on directmessages.parent_listing_id = listings.id
		where listings.creatorID = $1
		`, userID)
	if err != nil {
		fmt.Println(err)
		writeJsonERR(w, 500, "Unknown error!!!")
		return
	}

	directMessages := make([]DirectMessage, 0)
	for rows.Next() {
		dm := DirectMessage{}
		// dm.ParentListingID = parentListingID
		rows.Scan(
			&dm.ID,
			&dm.UserID,
			&dm.Content,
			&dm.ParentListingID,
			&dm.CreateDate,
			&dm.EndDate)
		directMessages = append(directMessages, dm)
	}
	retVal := struct {
		Error          string
		DirectMessages []DirectMessage
	}{"", directMessages}
	data, err := json.Marshal(retVal)

	if err != nil {
		writeJsonERR(w, 500, "Json error!")
		return
	}
	w.Write(data)
}

// Get user by ID
func userByID(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	auth := &struct {
		SessionID string
	}{}

	if err := decodeJSON(w, r, auth); err != nil {
		writeJsonERR(w, 400, "Invalid JSON")
		return
	}

	// Check that authentication is working.
	if _, ok, err := CheckSessionsKey(auth.SessionID); !ok || err != nil {
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
	if _, ok, err := CheckSessionsKey(auth.SessionID); !ok || err != nil {
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
