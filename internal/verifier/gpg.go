package verifier

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gimme-that-iso/internal/util"
)

// VerifySignature verifies a file's signature using a downloaded GPG key.
// It takes the URL of the GPG key, the local path to the signature file, and the local path to the file to be verified.
func VerifySignature(keyURL, sigPath, filePath string) error {
	// Create a temporary directory for GPG
	tmpDir, err := os.MkdirTemp("", "gpg-verify-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// --- Download GPG key ---
	keyBytes, err := util.DownloadFile(keyURL)
	if err != nil {
		return fmt.Errorf("failed to download GPG key: %w", err)
	}
	keyPath := filepath.Join(tmpDir, "key.asc")
	if err := os.WriteFile(keyPath, keyBytes, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	// --- Execute GPG commands ---
	gpgHomeDir := filepath.Join(tmpDir, ".gnupg")
	if err := os.Mkdir(gpgHomeDir, 0700); err != nil {
		return fmt.Errorf("failed to create gpg home: %w", err)
	}

	// Import the key
	importCmd := exec.Command("gpg", "--homedir", gpgHomeDir, "--import", keyPath)
	if output, err := importCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gpg import failed: %w\nOutput: %s", err, string(output))
	}

	// Verify the signature
	verifyCmd := exec.Command("gpg", "--homedir", gpgHomeDir, "--verify", sigPath, filePath)
	if output, err := verifyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gpg verify failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// verifyEmbeddedSignatureWithGPGV verifies a file with an embedded PGP signature (cleartext-signed message)
func verifyEmbeddedSignatureWithGPGV(keyURL, filePath string) error {
	// Create a temporary directory for GPG
	tmpDir, err := os.MkdirTemp("", "gpg-verify-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download GPG key
	keyBytes, err := util.DownloadFile(keyURL)
	if err != nil {
		return fmt.Errorf("failed to download GPG key: %w", err)
	}
	keyPath := filepath.Join(tmpDir, "key.asc")
	if err := os.WriteFile(keyPath, keyBytes, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	// Create GPG home directory
	gpgHomeDir := filepath.Join(tmpDir, ".gnupg")
	if err := os.Mkdir(gpgHomeDir, 0700); err != nil {
		return fmt.Errorf("failed to create gpg home: %w", err)
	}

	// Import the key
	importCmd := exec.Command("gpg", "--homedir", gpgHomeDir, "--import", keyPath)
	if output, err := importCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gpg import failed: %w\nOutput: %s", err, string(output))
	}

	// For cleartext-signed messages, verify with just the file path
	verifyCmd := exec.Command("gpg", "--homedir", gpgHomeDir, "--verify", filePath)
	if output, err := verifyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gpg verify failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}
