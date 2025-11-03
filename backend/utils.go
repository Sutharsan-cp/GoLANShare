package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/robfig/cron/v3"
)

func hashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("Error opening file for hashing: %v", err)
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Printf("Error hashing file: %v", err)
		return ""
	}
	
	return hex.EncodeToString(h.Sum(nil))
}

func startExpiryCron() {
	c := cron.New()
	
	_, err := c.AddFunc("@daily", func() {
		log.Println("Running expiry cleanup...")
		files := getAllFiles()
		now := time.Now()
		deletedCount := 0
		
		for _, f := range files {
			if now.After(f.Expiry) {
				filePath := filepath.Join(config.UploadPath, f.Name)
				if err := os.Remove(filePath); err != nil {
					if !os.IsNotExist(err) {
						log.Printf("Error deleting expired file %s: %v", f.Name, err)
					}
				} else {
					log.Printf("Deleted expired file: %s", f.Name)
				}
				
				if err := deleteFile(f.Name); err != nil {
					log.Printf("Error deleting expired file entry %s: %v", f.Name, err)
				} else {
					deletedCount++
				}
			}
		}
		
		log.Printf("Expiry cleanup completed. Deleted %d files.", deletedCount)
	})
	
	if err != nil {
		log.Printf("Error scheduling expiry cron: %v", err)
		return
	}
	
	c.Start()
	log.Println("Expiry cron started - will run daily")
}

// Utility function to get file extension
func getFileExtension(filename string) string {
	return filepath.Ext(filename)
}

// Utility function to format file size for display
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}