package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"castafiore-backend/internal/config"
	"castafiore-backend/internal/database"
	"castafiore-backend/internal/library"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting Castafiore Library Scanner...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	// Get music path from config file
	musicPath, err := getMusicPath()
	if err != nil {
		log.Fatalf("Failed to get music path: %v", err)
	}

	log.Printf("Music path configured: %s", musicPath)

	// Check if music path exists
	if _, err := os.Stat(musicPath); os.IsNotExist(err) {
		log.Fatalf("Music path does not exist: %s", musicPath)
	}

	// Create scanner
	scanner := library.NewScanner(db.DB)

	// Get stats before scan
	statsBefore, err := scanner.GetScanStats()
	if err != nil {
		log.Printf("Warning: Could not get stats before scan: %v", err)
	} else {
		log.Printf("Library stats before scan - Artists: %d, Albums: %d, Songs: %d",
			statsBefore["artists"], statsBefore["albums"], statsBefore["songs"])
	}

	// Scan the library
	log.Println("Starting library scan...")
	if err := scanner.ScanLibrary(musicPath); err != nil {
		log.Fatalf("Library scan failed: %v", err)
	}

	// Get stats after scan
	statsAfter, err := scanner.GetScanStats()
	if err != nil {
		log.Printf("Warning: Could not get stats after scan: %v", err)
	} else {
		log.Printf("Library stats after scan - Artists: %d, Albums: %d, Songs: %d",
			statsAfter["artists"], statsAfter["albums"], statsAfter["songs"])
	}

	log.Println("Library scan completed successfully!")
}

// getMusicPath reads the music path from the config file
func getMusicPath() (string, error) {
	configFile := "config/music_path.txt"
	content, err := os.ReadFile(configFile)
	if err != nil {
		return "", fmt.Errorf("failed to read music path config: %v", err)
	}

	musicPath := string(content)
	musicPath = strings.TrimSpace(musicPath)

	if musicPath == "" {
		return "", fmt.Errorf("music path is empty in config file")
	}

	return musicPath, nil
}
