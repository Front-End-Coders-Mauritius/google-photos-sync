package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.Println("Opening database...")

	db, err := sql.Open("sqlite3", "./timeliner_repo/index.db")
	if err != nil {
		log.Fatalf("Open database: %v", err)
	}
	defer db.Close()

	log.Println("Quering database...")
	rows, err := db.Query(`
		select c.name, i.data_file from items i
			inner join collection_items ci on ci.item_id = i.id
			inner join collections c on ci.collection_id = c.id
	`)
	if err != nil {
		log.Fatalf("Query database: %v", err)
	}
	defer rows.Close()

	log.Println("Looping over rows...")
	photos := make(map[string][]string)
	for rows.Next() {
		var (
			albumName string
			photoPath string
		)
		if err = rows.Scan(&albumName, &photoPath); err != nil {
			log.Fatal(err)
		}

		photos[albumName] = append(photos[albumName], photoPath)
	}
	if err = rows.Err(); err != nil {
		log.Fatalf("Loop over rows: %v", err)
	}

	log.Println("Marshaling photos...")
	jsonStr, err := json.Marshal(photos)
	if err != nil {
		log.Fatalf("Marshal photos: %v", err)
	}

	log.Println("Writing to file...")
	if err = ioutil.WriteFile("./index.json", jsonStr, 0644); err != nil {
		log.Fatalf("Write json file: %v", err)
	}
}
