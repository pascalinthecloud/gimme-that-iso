package verifier

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gimme-that-iso/internal/config"
	"gimme-that-iso/internal/util"
)

// VerifyISO performs GPG and checksum verification for a downloaded ISO.
func VerifyISO(workerID int, iso config.ISO, isoPath string) error {
	log.Printf("[Worker %d] [%s] Starting verification.", workerID, iso.Name)
	// Alpine-style verification: signature is for the ISO itself
	if strings.HasSuffix(iso.SignatureURL, ".iso.asc") {
		return verifyAlpineStyle(workerID, iso, isoPath)
	}

	// Debian-style verification: signature is for the checksum file
	return verifyDebianStyle(workerID, iso, isoPath)
}

func verifyAlpineStyle(workerID int, iso config.ISO, isoPath string) error {
	log.Printf("[Worker %d] [%s] Verifying GPG signature of ISO file...", workerID, iso.Name)
	// Download signature file
	sigBytes, err := util.DownloadFile(iso.SignatureURL)
	if err != nil {
		return fmt.Errorf("failed to download signature file: %w", err)
	}
	tmpSigFile, err := os.CreateTemp("", "signature-*.asc")
	if err != nil {
		return fmt.Errorf("failed to create temp signature file: %w", err)
	}
	defer os.Remove(tmpSigFile.Name())
	if _, err := tmpSigFile.Write(sigBytes); err != nil {
		return fmt.Errorf("failed to write to temp signature file: %w", err)
	}
	tmpSigFile.Close()

	// Verify the ISO file against its signature
	if err := VerifySignature(iso.GPGKeyURL, tmpSigFile.Name(), isoPath); err != nil {
		return fmt.Errorf("GPG signature verification failed for ISO: %w", err)
	}
	log.Printf("[Worker %d] [%s] GPG signature of ISO verified successfully.", workerID, iso.Name)

	// Now verify the checksum
	log.Printf("[Worker %d] [%s] Verifying checksum...", workerID, iso.Name)
	checksumBytes, err := util.DownloadFile(iso.ChecksumFileURL)
	if err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}
	checksumMap, err := ParseChecksumFile(bytes.NewReader(checksumBytes))
	if err != nil {
		return fmt.Errorf("failed to parse checksum file: %w", err)
	}

	isoFilename := filepath.Base(isoPath)
	expectedChecksum, ok := checksumMap[isoFilename]
	if !ok {
		return fmt.Errorf("checksum not found for file %s", isoFilename)
	}

	calculatedChecksum, err := CalculateChecksum(isoPath, "sha256") // Alpine uses sha256
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if calculatedChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, calculatedChecksum)
	}

	log.Printf("[Worker %d] [%s] Checksum verified successfully.", workerID, iso.Name)
	return nil
}

func verifyDebianStyle(workerID int, iso config.ISO, isoPath string) error {
	log.Printf("[Worker %d] [%s] Verifying GPG signature of checksum file...", workerID, iso.Name)
	// Download signature and checksum files
	sigBytes, err := util.DownloadFile(iso.SignatureURL)
	if err != nil {
		return fmt.Errorf("failed to download signature file: %w", err)
	}
	checksumBytes, err := util.DownloadFile(iso.ChecksumFileURL)
	if err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}

	// Create temporary files for verification
	tmpSigFile, err := os.CreateTemp("", "signature-*.sig")
	if err != nil {
		return fmt.Errorf("failed to create temp signature file: %w", err)
	}
	defer os.Remove(tmpSigFile.Name())
	if _, err := tmpSigFile.Write(sigBytes); err != nil {
		return fmt.Errorf("failed to write to temp signature file: %w", err)
	}
	tmpSigFile.Close()

	tmpChecksumFile, err := os.CreateTemp("", "checksums-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp checksum file: %w", err)
	}
	defer os.Remove(tmpChecksumFile.Name())
	if _, err := tmpChecksumFile.Write(checksumBytes); err != nil {
		return fmt.Errorf("failed to write to temp checksum file: %w", err)
	}
	tmpChecksumFile.Close()

	// Verify the checksum file against its signature
	if err := VerifySignature(iso.GPGKeyURL, tmpSigFile.Name(), tmpChecksumFile.Name()); err != nil {
		return fmt.Errorf("GPG signature verification failed for checksum file: %w", err)
	}
	log.Printf("[Worker %d] [%s] GPG signature of checksum file verified successfully.", workerID, iso.Name)

	// Now verify the ISO checksum
	log.Printf("[Worker %d] [%s] Verifying ISO checksum...", workerID, iso.Name)
	checksums, err := ParseChecksumFile(bytes.NewReader(checksumBytes))
	if err != nil {
		return fmt.Errorf("failed to parse checksum file: %w", err)
	}

	isoFilename := filepath.Base(isoPath)
	expectedChecksum, ok := checksums[isoFilename]
	if !ok {
		return fmt.Errorf("checksum not found for file %s", isoFilename)
	}

	hashAlgo := "sha512" // Default for the debian example
	if strings.Contains(strings.ToLower(iso.ChecksumFileURL), "sha256") {
		hashAlgo = "sha256"
	}

	calculatedChecksum, err := CalculateChecksum(isoPath, hashAlgo)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if calculatedChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, calculatedChecksum)
	}

	log.Printf("[Worker %d] [%s] Checksum verified successfully.", workerID, iso.Name)
	return nil
}