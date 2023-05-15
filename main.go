package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS events_log (
		code text not null,
		description text not null,
		metadata json not null,
		created_at timestamp(6) not null default current_timestamp(6)
	)`,
}

// Event is a simple struct that represents an event.
type Event struct {
	Code        string
	Description string
	Metadata    Metadata
	CreatedAt   time.Time
}

func (e Event) String() string {
	return fmt.Sprintf("Code: %s\nDescription: %s\nMetadata: %v\nCreatedAt: %v\n", e.Code, e.Description, e.Metadata, e.CreatedAt.Format(time.RFC3339))
}

// KV is a simple key value struct
type KV struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

// Value implements the driver Valuer interface for KV.
func (kv KV) Value() (driver.Value, error) {
	return json.Marshal(kv)
}

// Scan implements the Scanner interface for KV.
func (kv *KV) Scan(src any) error {
	switch src := src.(type) {
	case []byte:
		return json.Unmarshal(src, kv)
	default:
		return fmt.Errorf("incompatible type for KV (%T)", src)
	}
}

// Metadata is a custom type that implements the driver.Valuer and sql.Scanner
type Metadata []KV

// Value implements the driver Valuer interface.
func (m Metadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the Scanner interface.
func (m *Metadata) Scan(src any) error {
	switch src := src.(type) {
	case []byte:
		return json.Unmarshal(src, m)
	default:
		return fmt.Errorf("incompatible type for Metadata (%T)", src)
	}
}

func main() {
	db := setupDB()

	// Create some events
	events := []Event{
		{
			Code:        "golangdorset.meetup.begin",
			Description: "Golang Dorset Meetup has begun.",
			Metadata: Metadata{
				KV{Key: "location", Val: "Bournemouth"},
				KV{Key: "attendees", Val: "10"},
				KV{Key: "organiser", Val: "Golang Dorset"},
				KV{Key: "talks", Val: "2"},
			},
		},
		{
			Code:        "golangdorset.meetup.end",
			Description: "Golang Dorset Meetup has finished.",
			Metadata: Metadata{
				KV{Key: "location", Val: "Bournemouth"},
				KV{Key: "attendees", Val: "10"},
				KV{Key: "organiser", Val: "Golang Dorset"},
				KV{Key: "talks", Val: "2"},
			},
		},
	}

	// Insert the events into the database
	const stmt = `
INSERT INTO events_log (code, description, metadata)
     VALUES (?, ?, ?)`

	for _, event := range events {
		_, err := db.Exec(stmt, event.Code, event.Description, event.Metadata)
		if err != nil {
			log.Fatalf("Error inserting event: %v", err)
		}
	}

	// Query the database for the first event found
	const query = `
SELECT code, description, metadata, created_at
  FROM events_log
 WHERE code = ?
 LIMIT 1`

	var e Event
	row := db.QueryRow(query, "golangdorset.meetup.begin")
	if err := row.Scan(&e.Code, &e.Description, &e.Metadata, &e.CreatedAt); err != nil {
		log.Fatalf("Error querying event: %v", err)
	}

	// Print the event
	fmt.Printf("%+v\n", e)

	// Query the database for all events where the location is Discord
	const query2 = `
SELECT code, description, metadata, created_at
  FROM events_log
 WHERE JSON_CONTAINS(metadata, ?)`
	rows, err := db.Query(query2, KV{Key: "location", Val: "Bournemouth"})
	if err != nil {
		log.Fatalf("Error querying events: %v", err)
	}
	defer rows.Close()

	// Print the events
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.Code, &e.Description, &e.Metadata, &e.CreatedAt); err != nil {
			log.Printf("Error scanning event: %v", err)
		}
		fmt.Printf("%+v\n", e)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating over events: %v", err)
	}
}

func setupDB() *sql.DB {
	db, err := sql.Open("mysql", "root:password@tcp(0.0.0.0:3306)/golangdorset?charset=utf8&parseTime=true")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	for i, v := range migrations {
		_, err := db.Exec(v)
		if err != nil {
			log.Fatalf("Error executing migration %d: %v", i, err)
		}
	}

	return db
}

// insertEvent inserts an event into the database.
func insertEvent(db *sql.DB, event Event) error {
	const stmt = `
INSERT INTO events_log (code, description, metadata)
     VALUES (?, ?, ?)`

	_, err := db.Exec(stmt, event.Code, event.Description, event.Metadata)
	if err != nil {
		return fmt.Errorf("error inserting event: %v", err)
	}
	return nil
}
