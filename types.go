// TODO: Separate routes out into other fuctions and so on.
package main

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/lib/pq"
)

type NullInt64 struct{ sql.NullInt64 }
type NullString struct{ sql.NullString }
type NullTime struct{ pq.NullTime }

func (s *NullString) UnmarshalJSON(data []byte) error {
	s.Valid = true
	s.String = string(data)
	return json.Unmarshal(data, &s.String)
}

func (s *NullInt64) UnmarshalJSON(data []byte) error {
	s.Valid = true
	val, err := strconv.Atoi(string(data))
	s.Int64 = int64(val)
	return err
}

func (s *NullTime) UnmarshalJSON(data []byte) error {
	s.Valid = true
	return s.Time.UnmarshalJSON(data)
}

func (s *NullString) Set(o string) {
	s.Valid = true
	s.String = o
}

func (s *NullInt64) Set(o int64) {
	s.Valid = true
	s.Int64 = o
}
func (s *NullTime) Set(o time.Time) {
	s.Valid = true
	s.Time = o
}
func (s NullString) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String)
}

func (s NullInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Int64)
}

func (s NullTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Time)
}

type User struct {
	Handle   NullString
	Email    NullString
	Bio      NullString
	ImageURL NullString
}

type Comment struct {
	ID              NullInt64
	UserID          NullInt64
	Content         NullString
	ParentListingID NullInt64
	CreateDate      NullTime
	EndDate         NullTime
}

type DirectMessage struct {
	ID              NullInt64
	UserID          NullInt64
	Content         NullString
	ParentListingID NullInt64
	CreateDate      NullTime
	EndDate         NullTime
}

type Listing struct {
	ListingID  NullInt64
	UserID     NullInt64
	Title      NullString
	Content    NullString
	Price      NullString
	Condition  NullString
	CreateDate NullTime
	EndDate    NullTime
}

type Book struct {
	Id          NullInt64
	Title       NullString
	Author      NullString
	Description NullString
	ImageUrl    NullString
	ISBN_13     NullString
	ISBN_10     NullString
}

type Course struct {
	Id          NullInt64
	CreatorID   NullInt64
	Title       NullString
	Description NullString
}
