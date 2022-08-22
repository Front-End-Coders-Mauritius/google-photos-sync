package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/disintegration/imaging"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nickalie/go-webpbin"
	"golang.org/x/sync/errgroup"
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

	var (
		photos = make(map[string][]string)
		g      = new(errgroup.Group)
		mu     sync.Mutex
	)
	log.Println("Looping over rows...")
	for rows.Next() {
		var (
			albumName string
			photoID   string
			photoPath string
		)
		if err = rows.Scan(&albumName, &photoID, &photoPath); err != nil {
			log.Printf("scan row: %v", err)
			continue
		}

		g.Go(func() error {
			fullPhotoPath := filepath.Join(timelinerRepo, photoPath)
			if _, err = os.Stat(fullPhotoPath); err != nil {
				// image not found so we skip processing it
				return nil
			}

			fullDstPhotoDir := filepath.Dir(filepath.Join(timelinerRepo, "processed", photoPath))
			fullProcessedPhotoPath := filepath.Join(fullDstPhotoDir, photoID+".webp")
			if _, err = os.Stat(fullProcessedPhotoPath); err == nil {
				// processed image exists so we skip processing it
				mu.Lock()
				photos[albumName] = append(photos[albumName], fullProcessedPhotoPath)
				mu.Unlock()

				return nil
			}

			img, err := imaging.Open(fullPhotoPath)
			if err != nil {
				return fmt.Errorf("open image '%s': %w", photoID, err)
			}

			if err = os.MkdirAll(fullDstPhotoDir, os.ModePerm); err != nil {
				return fmt.Errorf("create directory structure '%s': %w", photoID, err)
			}

			img = imaging.Fill(img, 1920, 1080, imaging.Center, imaging.Lanczos)

			f, err := os.Create(fullProcessedPhotoPath)
			if err != nil {
				return fmt.Errorf("create webp image '%s': %w", photoID, err)
			}

			if err = webpbin.Encode(f, img); err != nil {
				f.Close()
				return fmt.Errorf("save webp image '%s': %w", photoID, err)
			}

			if err = f.Close(); err != nil {
				return fmt.Errorf("close webp image '%s': %w", photoID, err)
			}

			mu.Lock()
			photos[albumName] = append(photos[albumName], fullProcessedPhotoPath)
			mu.Unlock()

			return nil
		})
	}
	if err = rows.Err(); err != nil {
		log.Fatalf("Loop over rows: %v", err)
	}

	if err = g.Wait(); err != nil {
		log.Printf("Multiple errors: %v\n", err)
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
