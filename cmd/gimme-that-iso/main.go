package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gimme-that-iso/internal/config"
	"gimme-that-iso/internal/downloader"
	"gimme-that-iso/internal/ui" // Import the new ui package
	"gimme-that-iso/internal/verifier"
)

// JobResult tracks the result of a download/verification job
type JobResult struct {
	Name                string
	URL                 string
	Status              string // "success", "download_failed", "verification_failed"
	FileSize            int64
	StartTime           time.Time
	EndTime             time.Time
	DownloadDuration    time.Duration
	VerifyDuration      time.Duration
	TotalDuration       time.Duration
	DestinationPath     string
	ErrorMessage        string
	VerificationSkipped bool // true if no verification URLs were provided
}

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
	results := make(chan JobResult, len(cfg.ISOs))
	var wg sync.WaitGroup

	// Start workers
	for w := 1; w <= *numWorkers; w++ {
		go worker(w, *downloadDir, jobs, results, &wg)
	}

	// Send jobs
	for _, iso := range cfg.ISOs {
		wg.Add(1)
		jobs <- iso
	}
	close(jobs)

	wg.Wait()
	close(results)

	// Collect and display results
	var jobResults []JobResult
	for result := range results {
		jobResults = append(jobResults, result)
	}

	printSummary(jobResults)
}

func worker(id int, downloadDir string, jobs <-chan config.ISO, results chan<- JobResult, wg *sync.WaitGroup) {
	for iso := range jobs {
		defer wg.Done()

		result := JobResult{
			Name:      iso.Name,
			URL:       iso.URL,
			StartTime: time.Now(),
		}

		log.Printf("[Worker %d] [%s] Processing job.", id, iso.Name)
		destPath := filepath.Join(downloadDir, filepath.Base(iso.URL))
		result.DestinationPath = destPath

		// Download phase
		downloadStart := time.Now()
		err := downloader.DownloadFile(id, iso.Name, iso.URL, destPath, true)
		result.DownloadDuration = time.Since(downloadStart)

		if err != nil {
			result.Status = "download_failed"
			result.ErrorMessage = err.Error()
			log.Printf("[Worker %d] [%s] ERROR: Failed to download: %v", id, iso.Name, err)
			result.EndTime = time.Now()
			result.TotalDuration = time.Since(result.StartTime)
			results <- result
			continue
		}

		// Get file size
		if fileInfo, err := os.Stat(destPath); err == nil {
			result.FileSize = fileInfo.Size()
		}

		// Verification phase
		verifyStart := time.Now()
		err = verifier.VerifyISO(id, iso, destPath)
		result.VerifyDuration = time.Since(verifyStart)

		// Check if verification was skipped (no error but no verification URLs)
		if iso.GPGKeyURL == "" && iso.SignatureURL == "" && iso.ChecksumFileURL == "" {
			result.VerificationSkipped = true
		}

		if err != nil {
			result.Status = "verification_failed"
			result.ErrorMessage = err.Error()
			log.Printf("[Worker %d] [%s] ERROR: Verification failed: %v", id, iso.Name, err)
		} else {
			result.Status = "success"
			log.Printf("[Worker %d] [%s] Successfully verified.", id, iso.Name)
		}

		result.EndTime = time.Now()
		result.TotalDuration = time.Since(result.StartTime)
		results <- result
	}
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(bytes)
	unitIndex := 0

	for value >= 1024 && unitIndex < len(units)-1 {
		value /= 1024
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", value, units[unitIndex])
}

// formatDuration converts duration to human-readable format
func formatDuration(d time.Duration) string {
	hours := d / time.Hour
	d %= time.Hour
	minutes := d / time.Minute
	d %= time.Minute
	seconds := d / time.Second

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// printSummary prints a formatted summary table of all download/verification results
func printSummary(results []JobResult) {
	fmt.Println("\n" + strings.Repeat("=", 128))
	fmt.Println("DOWNLOAD & VERIFICATION SUMMARY")
	fmt.Println(strings.Repeat("=", 128))
	fmt.Println()

	if len(results) == 0 {
		fmt.Println("No jobs completed.")
		return
	}

	// Print header
	fmt.Printf("%-28s | %-11s | %-12s | %-10s | %-8s | %-10s | %-11s | %-16s\n",
		"ISO Name", "Status", "File Size", "Download", "Verify", "Total", "Speed", "Verification")
	fmt.Println(strings.Repeat("-", 128))

	totalSize := int64(0)
	successCount := 0
	failedCount := 0

	for _, result := range results {
		statusStr := result.Status
		if result.Status == "success" {
			statusStr = "✓ SUCCESS"
			successCount++
		} else if result.Status == "download_failed" {
			statusStr = "✗ DL FAIL"
			failedCount++
		} else if result.Status == "verification_failed" {
			statusStr = "✗ VER FAIL"
			failedCount++
		}

		fileSize := formatBytes(result.FileSize)
		downloadTime := formatDuration(result.DownloadDuration)
		verifyTime := formatDuration(result.VerifyDuration)
		totalTime := formatDuration(result.TotalDuration)

		// Calculate download speed in MB/s
		var speed string
		if result.DownloadDuration > 0 {
			speedMbs := float64(result.FileSize) / result.DownloadDuration.Seconds() / (1024 * 1024)
			speed = fmt.Sprintf("%.1f MB/s", speedMbs)
		} else {
			speed = "N/A"
		}

		var verificationStatus string
		if result.VerificationSkipped {
			verificationStatus = "Skipped"
		} else if result.Status == "success" {
			verificationStatus = "✓ Verified"
		} else if result.Status == "verification_failed" {
			verificationStatus = "✗ Failed"
		} else {
			verificationStatus = "N/A"
		}

		fmt.Printf("%-28s | %-11s | %-12s | %-10s | %-8s | %-10s | %-11s | %-16s\n",
			truncateString(result.Name, 26), statusStr, fileSize, downloadTime, verifyTime, totalTime, speed, verificationStatus)

		if result.Status == "success" {
			totalSize += result.FileSize
		} else {
			fmt.Printf("  └─ Error: %s\n", result.ErrorMessage)
		}
	}

	fmt.Println(strings.Repeat("-", 128))

	// Print summary statistics
	fmt.Printf("\nSummary: %d succeeded, %d failed | Total Downloaded: %s\n",
		successCount, failedCount, formatBytes(totalSize))

	// Calculate total duration
	if len(results) > 0 {
		minStart := results[0].StartTime
		maxEnd := results[0].EndTime
		for _, r := range results {
			if r.StartTime.Before(minStart) {
				minStart = r.StartTime
			}
			if r.EndTime.After(maxEnd) {
				maxEnd = r.EndTime
			}
		}
		totalDuration := maxEnd.Sub(minStart)
		fmt.Printf("Total Execution Time: %s\n", formatDuration(totalDuration))
	}

	fmt.Println(strings.Repeat("=", 128) + "\n")
}

// truncateString truncates a string to a maximum length with ellipsis if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
