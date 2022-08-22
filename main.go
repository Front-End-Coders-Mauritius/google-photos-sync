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
	"github.com/panjf2000/ants"
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

	// will force download of any webp binary
	webpbin.NewCWebP().BinWrapper.Run()

	p, err := ants.NewPool(25)
	if err != nil {
		log.Fatalf("New pool: %v", err)
	}

	var (
		errs   error
		photos = make(map[string][]string)
		mu     sync.Mutex
		wg     sync.WaitGroup
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

		wg.Add(1)
		p.Submit(func() {
			defer wg.Done()

			fullPhotoPath := filepath.Join(timelinerRepo, photoPath)
			if _, err = os.Stat(fullPhotoPath); err != nil {
				// image not found so we skip processing it
				return
			}

			fullDstPhotoDir := filepath.Dir(filepath.Join(timelinerRepo, "processed", photoPath))
			fullProcessedPhotoPath := filepath.Join(fullDstPhotoDir, photoID+".webp")
			if _, err = os.Stat(fullProcessedPhotoPath); err == nil {
				// processed image exists so we skip processing it
				mu.Lock()
				photos[albumName] = append(photos[albumName], fullProcessedPhotoPath)
				mu.Unlock()

				return
			}

			img, err := imaging.Open(fullPhotoPath)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf("open image '%s': %w", photoPath, err))

				return
			}

			if err = os.MkdirAll(fullDstPhotoDir, os.ModePerm); err != nil {
				errs = multierr.Append(errs, fmt.Errorf("create directory structure '%s': %w", photoPath, err))

				return
			}

			bg := imaging.Clone(img)
			bg = imaging.Fill(bg, 1920, 1080, imaging.Center, imaging.Lanczos)
			bg = imaging.Blur(bg, 5)

			if img.Bounds().Dx() >= img.Bounds().Dy() {
				img = imaging.Resize(img, 1920, 0, imaging.Lanczos)
			} else {
				img = imaging.Resize(img, 0, 1080, imaging.Lanczos)
			}
			img = imaging.PasteCenter(bg, img)

			f, err := os.Create(fullProcessedPhotoPath)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf("create webp image '%s': %w", photoPath, err))

				return
			}

			if err = webpbin.Encode(f, img); err != nil {
				f.Close()
				errs = multierr.Append(errs, fmt.Errorf("save webp image '%s': %w", photoPath, err))

				return
			}

			if err = f.Close(); err != nil {
				errs = multierr.Append(errs, fmt.Errorf("close webp image '%s': %w", photoPath, err))

				return
			}

			mu.Lock()
			photos[albumName] = append(photos[albumName], fullProcessedPhotoPath)
			mu.Unlock()

			return
		})
	}
	if err = rows.Err(); err != nil {
		log.Fatalf("Loop over rows: %v", err)
	}

	wg.Wait()

	if errs != nil {
		log.Printf("Multiple errors: %v\n", errs)
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
