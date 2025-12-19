package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gimme-that-iso/internal/config"
	"gimme-that-iso/internal/downloader"
	"gimme-that-iso/internal/ui" // Import the new ui package
	"gimme-that-iso/internal/verifier"
)

func main() {
	ui.PrintBanner() // Call the banner function at the start

	// Configure the logger to use a standard format with date and time
	log.SetFlags(log.LstdFlags)
	
	log.Println("gimme-that-iso starting...")

	// Command-line flags
	downloadDir := flag.String("download-dir", "downloads", "Directory to save downloaded ISOs")
	numWorkers := flag.Int("workers", 4, "Number of concurrent download workers")
	flag.Parse()

	// Create the downloads directory if it doesn't exist
	if err := os.MkdirAll(*downloadDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create download directory: %v", err)
	}

	cfg, err := config.LoadConfig("isos.json")
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully. Starting up to %d workers...", *numWorkers)

	jobs := make(chan config.ISO, len(cfg.ISOs))
	var wg sync.WaitGroup

	// Start workers
	for w := 1; w <= *numWorkers; w++ {
		go worker(w, *downloadDir, jobs, &wg)
	}

	// Send jobs
	for _, iso := range cfg.ISOs {
		wg.Add(1)
		jobs <- iso
	}
	close(jobs)

	wg.Wait()
	log.Println("All tasks completed.")
}

func worker(id int, downloadDir string, jobs <-chan config.ISO, wg *sync.WaitGroup) {
	for iso := range jobs {
		defer wg.Done()
		log.Printf("[Worker %d] [%s] Processing job.", id, iso.Name)
		destPath := filepath.Join(downloadDir, filepath.Base(iso.URL))
		
		err := downloader.DownloadFile(id, iso.Name, iso.URL, destPath, true) // Always show detailed progress
		if err != nil {
			log.Printf("[Worker %d] [%s] ERROR: Failed to download: %v", id, iso.Name, err)
			continue
		}

		// Verification will now also print prefixed logs
		if err := verifier.VerifyISO(id, iso, destPath); err != nil {
			log.Printf("[Worker %d] [%s] ERROR: Verification failed: %v", id, iso.Name, err)
		} else {
			log.Printf("[Worker %d] [%s] Successfully verified.", id, iso.Name)
		}
	}
}
