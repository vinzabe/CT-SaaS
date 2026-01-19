package torrent

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateZipFromFiles creates a zip archive from a list of files
func CreateZipFromFiles(downloadDir, torrentName string, files []string) (string, int64, error) {
	// Create zip file path
	zipName := sanitizeFileName(torrentName) + ".zip"
	zipPath := filepath.Join(downloadDir, zipName)
	
	// Create zip file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()
	
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	
	// Add each file to the zip
	for _, filePath := range files {
		fullPath := filepath.Join(downloadDir, filePath)
		
		// Security check
		if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(downloadDir)) {
			continue
		}
		
		// Check if file exists
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		
		// Skip directories
		if info.IsDir() {
			continue
		}
		
		// Create zip entry
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			continue
		}
		
		// Use the relative path as the name in the zip
		header.Name = filePath
		header.Method = zip.Deflate
		
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			continue
		}
		
		// Open and copy file
		file, err := os.Open(fullPath)
		if err != nil {
			continue
		}
		
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			continue
		}
	}
	
	// Close the zip writer to flush data
	zipWriter.Close()
	zipFile.Close()
	
	// Get zip file size
	zipInfo, err := os.Stat(zipPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to stat zip file: %w", err)
	}
	
	return zipName, zipInfo.Size(), nil
}

// sanitizeFileName removes invalid characters from filename
func sanitizeFileName(name string) string {
	// Replace invalid characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	
	// Limit length
	if len(result) > 200 {
		result = result[:200]
	}
	
	// Remove leading/trailing spaces and dots
	result = strings.Trim(result, " .")
	
	if result == "" {
		result = "download"
	}
	
	return result
}
