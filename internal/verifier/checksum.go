package verifier

import (
	"bufio"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

// ParseChecksumFile parses a checksum file (e.g., SHA512SUMS) and returns a map of filenames to their checksums.
func ParseChecksumFile(reader io.Reader) (map[string]string, error) {
	checksums := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// Expect format: CHECKSUM  FILENAME
		parts := strings.Fields(line)
		if len(parts) == 2 {
			checksums[parts[1]] = parts[0]
		}
	}
	return checksums, scanner.Err()
}

// CalculateChecksum calculates the checksum of a file using the provided hash algorithm.
func CalculateChecksum(filePath string, hashAlgo string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var h hash.Hash
	switch hashAlgo {
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", hashAlgo)
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
