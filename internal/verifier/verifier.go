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
// If no verification URLs are provided, it skips verification.
func VerifyISO(workerID int, iso config.ISO, isoPath string) error {
	// Check if verification is configured
	if iso.GPGKeyURL == "" && iso.SignatureURL == "" && iso.ChecksumFileURL == "" {
		log.Printf("[Worker %d] [%s] No verification URLs provided, skipping verification.", workerID, iso.Name)
		return nil
	}

	log.Printf("[Worker %d] [%s] Starting verification.", workerID, iso.Name)

	// Use explicit verification type from config
	switch iso.VerificationType {
	case "iso_signed":
		return verifyISOSigned(workerID, iso, isoPath)
	case "checksum_signed":
		return verifyChecksumSigned(workerID, iso, isoPath)
	case "checksum_embedded":
		return verifyChecksumEmbedded(workerID, iso, isoPath)
	default:
		// Fallback to old behavior for backwards compatibility
		if iso.SignatureURL != "" && strings.HasSuffix(iso.SignatureURL, ".iso.asc") {
			return verifyISOSigned(workerID, iso, isoPath)
		}
		if iso.ChecksumFileURL != "" || iso.SignatureURL != "" {
			return verifyChecksumSigned(workerID, iso, isoPath)
		}
		// No verification URLs found
		log.Printf("[Worker %d] [%s] No verification URLs provided, skipping verification.", workerID, iso.Name)
		return nil
	}
}

func verifyISOSigned(workerID int, iso config.ISO, isoPath string) error {
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

func verifyChecksumSigned(workerID int, iso config.ISO, isoPath string) error {
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

// verifyChecksumEmbedded handles Fedora-style verification where the GPG signature is embedded in the checksum file
func verifyChecksumEmbedded(workerID int, iso config.ISO, isoPath string) error {
	log.Printf("[Worker %d] [%s] Verifying embedded GPG signature in checksum file...", workerID, iso.Name)

	// Download the signed checksum file (cleartext-signed message)
	checksumBytes, err := util.DownloadFile(iso.ChecksumFileURL)
	if err != nil {
		return fmt.Errorf("failed to download checksum file: %w", err)
	}

	// Create temporary file for the signed checksum
	tmpSignedFile, err := os.CreateTemp("", "signed-checksum-*.txt")
	if err != nil {
		return fmt.Errorf("failed to create temp signed file: %w", err)
	}
	defer os.Remove(tmpSignedFile.Name())
	if _, err := tmpSignedFile.Write(checksumBytes); err != nil {
		return fmt.Errorf("failed to write to temp signed file: %w", err)
	}
	tmpSignedFile.Close()

	// Verify the embedded signature using GPG with the downloaded key
	if err := verifyEmbeddedSignatureWithGPGV(iso.GPGKeyURL, tmpSignedFile.Name()); err != nil {
		return fmt.Errorf("GPG signature verification failed: %w", err)
	}
	log.Printf("[Worker %d] [%s] GPG signature verified successfully.", workerID, iso.Name)

	// Extract plain checksums from the cleartext-signed message
	plainChecksums := extractPlainChecksums(checksumBytes)
	checksumMap, err := ParseChecksumFile(bytes.NewReader(plainChecksums))
	if err != nil {
		return fmt.Errorf("failed to parse checksum data: %w", err)
	}

	// Verify the ISO checksum
	log.Printf("[Worker %d] [%s] Verifying ISO checksum...", workerID, iso.Name)
	isoFilename := filepath.Base(isoPath)
	expectedChecksum, ok := checksumMap[isoFilename]
	if !ok {
		return fmt.Errorf("checksum not found for file %s", isoFilename)
	}

	calculatedChecksum, err := CalculateChecksum(isoPath, "sha256")
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if calculatedChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, calculatedChecksum)
	}

	log.Printf("[Worker %d] [%s] Checksum verified successfully.", workerID, iso.Name)
	return nil
} // extractPlainChecksums removes PGP signature headers and footers from a signed checksum file
// Also converts Fedora's "SHA256 (filename) = hash" format to standard "hash filename" format
func extractPlainChecksums(signedData []byte) []byte {
	lines := strings.Split(string(signedData), "\n")
	var plainLines []string
	inBody := false            // We're between "Hash: SHA256" and "-----BEGIN PGP SIGNATURE-----"
	var pendingChecksum string // For multi-line checksums
	var pendingFilename string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip PGP header
		if strings.HasPrefix(trimmed, "-----BEGIN PGP SIGNED MESSAGE-----") {
			continue
		}
		// After Hash declaration, we're in the body
		if strings.HasPrefix(trimmed, "Hash:") {
			inBody = true
			continue
		}
		// Stop at signature block
		if strings.HasPrefix(trimmed, "-----BEGIN PGP SIGNATURE-----") {
			inBody = false
			// Flush any pending checksum
			if pendingFilename != "" && pendingChecksum != "" {
				plainLines = append(plainLines, pendingChecksum+"  "+pendingFilename)
			}
			break
		}

		// Skip other PGP headers
		if strings.HasPrefix(trimmed, "-----") {
			continue
		}

		// If we're in the body, process checksums
		if inBody {
			// Skip comment lines (starting with #)
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			// Skip empty lines
			if trimmed == "" {
				continue
			}

			// Convert Fedora format "SHA256 (filename) = hash" to standard "hash filename"
			if strings.Contains(trimmed, "SHA256") && strings.Contains(trimmed, "=") && strings.HasPrefix(trimmed, "SHA256") {
				// Flush previous checksum if exists
				if pendingFilename != "" && pendingChecksum != "" {
					plainLines = append(plainLines, pendingChecksum+"  "+pendingFilename)
				}

				// Parse: SHA256 (Fedora-Workstation-Live-43-1.6.x86_64.iso) = 2a4a16c009...
				parts := strings.Split(trimmed, "=")
				if len(parts) == 2 {
					hashPart := strings.TrimSpace(parts[1])
					filenamePart := strings.TrimSpace(parts[0])
					// Extract filename from "SHA256 (filename)"
					if strings.HasPrefix(filenamePart, "SHA256") {
						filenamePart = strings.TrimPrefix(filenamePart, "SHA256")
						filenamePart = strings.TrimSpace(filenamePart)
						filenamePart = strings.TrimPrefix(filenamePart, "(")
						filenamePart = strings.TrimSuffix(filenamePart, ")")
						pendingChecksum = hashPart
						pendingFilename = filenamePart
						continue
					}
				}
			} else if pendingFilename != "" && trimmed != "" {
				// This is a continuation of a multi-line hash
				pendingChecksum += trimmed
				continue
			} else {
				// Standard format: already in "hash filename" format
				plainLines = append(plainLines, line)
			}
		}
	}

	// Flush any final pending checksum
	if pendingFilename != "" && pendingChecksum != "" {
		plainLines = append(plainLines, pendingChecksum+"  "+pendingFilename)
	}

	return []byte(strings.Join(plainLines, "\n"))
}
