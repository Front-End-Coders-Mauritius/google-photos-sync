package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nickalie/go-webpbin"
	"go.uber.org/multierr"
)

const timelinerRepo = "./timeliner_repo"

func main() {
	log.Println("Opening database...")

	db, err := sql.Open("sqlite3", timelinerRepo+"/index.db")
	if err != nil {
		log.Fatalf("Open database: %v", err)
	}
	defer db.Close()

	log.Println("Quering database...")
	rows, err := db.Query(`
		select c.name, i.original_id, i.data_file from items i
			inner join collection_items ci on ci.item_id = i.id
			inner join collections c on ci.collection_id = c.id
	`)
	if err != nil {
		log.Fatalf("Query database: %v", err)
	}
	defer rows.Close()

	var errs error
	log.Println("Looping over rows...")
	photos := make(map[string][]string)
	for rows.Next() {
		var (
			albumName string
			photoID   string
			photoPath string
		)
		if err = rows.Scan(&albumName, &photoID, &photoPath); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("scan row: %w", err))
			continue
		}

		fullPhotoPath := filepath.Join(timelinerRepo, photoPath)
		if _, err := os.Stat(fullPhotoPath); err != nil {
			continue
		}

		img, err := imaging.Open(fullPhotoPath)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("open image: %w", err))
			continue
		}

		fullDstPhotoDir := filepath.Dir(filepath.Join(timelinerRepo, "processed", photoPath))
		if err := os.MkdirAll(fullDstPhotoDir, os.ModePerm); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("create directory structure: %w", err))
			continue
		}

		img = imaging.Fill(img, 1920, 1080, imaging.Center, imaging.Lanczos)

		fullProcessedPhotoPath := filepath.Join(fullDstPhotoDir, photoID+".webp")
		f, err := os.Create(fullProcessedPhotoPath)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("create webp image: %w", err))
			continue
		}

		if err := webpbin.Encode(f, img); err != nil {
			f.Close()
			errs = multierr.Append(errs, fmt.Errorf("save webp image: %w", err))
			continue
		}

		if err := f.Close(); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("close webp image: %w", err))
			continue
		}

		photos[albumName] = append(photos[albumName], fullProcessedPhotoPath)
	}
	if err = rows.Err(); err != nil {
		log.Fatalf("Loop over rows: %v", err)
	}

	if errs != nil {
		log.Fatalf("Multiple errors: %v", errs)
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
