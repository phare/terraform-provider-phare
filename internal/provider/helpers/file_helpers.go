package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LogoAllowedExtensions defines the allowed file extensions for logo uploads.
var LogoAllowedExtensions = []string{".jpg", ".jpeg", ".png", ".svg"}

// FaviconAllowedExtensions defines the allowed file extensions for favicon uploads.
var FaviconAllowedExtensions = []string{".png", ".svg"}

// ValidateImageFile checks if the file exists, is readable, and has an allowed extension.
func ValidateImageFile(path string, allowedExtensions []string) error {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("cannot access file %s: %w", path, err)
	}

	// Check if it's a regular file (not a directory)
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(path))
	validExt := false
	for _, allowed := range allowedExtensions {
		if ext == allowed {
			validExt = true
			break
		}
	}
	if !validExt {
		return fmt.Errorf("invalid file extension %q, allowed: %v", ext, allowedExtensions)
	}

	return nil
}

// ComputeFileHash computes the SHA256 hash of a file's contents.
func ComputeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file for hashing: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// OpenFileForUpload opens a file and returns a ReadCloser for uploading.
func OpenFileForUpload(path string) (io.ReadCloser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for upload: %w", err)
	}
	return file, nil
}
