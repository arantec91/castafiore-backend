package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"castafiore-backend/internal/config"
	"castafiore-backend/internal/library"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Testing improved scanner...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connection successful")

	// Get music path from config file
	configFile := "config/music_path.txt"
	musicPathBytes, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read music path from %s: %v", configFile, err)
	}

	musicPath := string(musicPathBytes)
	log.Printf("Music path: %s", musicPath)

	// Check if music directory exists
	if _, err := os.Stat(musicPath); os.IsNotExist(err) {
		log.Fatalf("Music directory does not exist: %s", musicPath)
	}

	// Test the scanner with improved error handling
	scanner := library.NewScanner(db)

	// Get the first few audio files to test with
	var testFiles []string
	err = filepath.Walk(musicPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(path)
			if ext == ".mp3" || ext == ".flac" || ext == ".m4a" || ext == ".ogg" {
				testFiles = append(testFiles, path)
				if len(testFiles) >= 5 { // Test with first 5 files
					return filepath.SkipDir
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking directory: %v", err)
		return
	}

	log.Printf("Found %d audio files in library", len(testFiles))

	if len(testFiles) == 0 {
		log.Println("No audio files found to test")
		return
	}

	// Test the full scanner with our improved error handling
	log.Println("Starting library scan with improved error handling...")
	err = scanner.ScanLibrary(musicPath)
	if err != nil {
		log.Printf("Scanner completed with error: %v", err)
		log.Println("Note: This might be expected if there are problematic files")
	} else {
		log.Println("Scanner completed successfully!")
	}

	// Get final statistics
	stats, err := scanner.GetScanStats()
	if err != nil {
		log.Printf("Error getting stats: %v", err)
	} else {
		log.Printf("Final library stats: %+v", stats)
	}

	log.Println("Test completed")
}
