package downloader

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

const (
	// ConnectTimeout is the maximum time to establish a connection
	ConnectTimeout = 30 * time.Second
	// MaxDownloadTime is the maximum allowed time for a complete download
	// (prevents hung downloads that never finish)
	MaxDownloadTime = 2 * time.Hour
	// IdleTimeout is how long to wait without progress before timing out
	IdleTimeout = 5 * time.Minute
)

// DownloadFile handles the download of a single file. If showProgress is true,
// it prints periodic, detailed log lines to standard output.
func DownloadFile(workerID int, isoName string, url string, destPath string, showProgress bool) error {
	// Create a context with a maximum download timeout
	ctx, cancel := context.WithTimeout(context.Background(), MaxDownloadTime)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	// Create a custom HTTP client with connection timeout
	client := &http.Client{
		Timeout: ConnectTimeout + MaxDownloadTime,
		Transport: &http.Transport{
			IdleConnTimeout: IdleTimeout,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer f.Close()

	if !showProgress {
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return fmt.Errorf("could not write to file: %w", err)
		}
		return nil
	}

	// --- Detailed Logging Implementation ---
	bar := progressbar.NewOptions64(
		resp.ContentLength,
		progressbar.OptionSetWriter(io.Discard),                   // Don't render the bar
		progressbar.OptionThrottle(2*time.Second),                 // Throttle state updates
		progressbar.OptionSetDescription(filepath.Base(destPath)), // Use filename as description
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowBytes(true),
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !bar.IsFinished() {
			time.Sleep(2 * time.Second)
			state := bar.State()

			speedMBs := state.KBsPerSecond / 1024.0 // Convert KB/s to MB/s

			var eta time.Duration
			// Use the library's built-in SecondsLeft for a more stable ETA
			eta = time.Duration(state.SecondsLeft) * time.Second

			log.Printf(
				"[Worker %d] [%s] Progress: %.0f%% (%s / %s) @ %.2f MB/s, ETA: %s",
				workerID,
				isoName,
				state.CurrentPercent*100,
				formatBytes(int64(state.CurrentBytes)),
				formatBytes(state.Max),
				speedMBs,
				formatDuration(eta),
			)
		}
	}()

	// Use io.MultiWriter to write to file and update the bar
	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		// Can't return error here easily without more channels,
		// but the outer function will catch the io.Copy error.
	}

	// Wait for the logging goroutine to print its last update
	wg.Wait()
	log.Printf("[Worker %d] [%s] Download complete.", workerID, isoName)

	return err // Return the error from io.Copy
}

// formatBytes converts bytes to a human-readable string (KB, MB, GB)
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats a duration into a simpler string like "1m5s"
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%dm%ds", m, s)
}
