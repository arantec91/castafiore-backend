// Test script to verify the favorites functionality
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"castafiore-backend/internal/auth"
	"castafiore-backend/internal/config"
	"castafiore-backend/internal/database"
	"castafiore-backend/internal/subsonic"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize services
	authService := auth.NewService(cfg.JWTSecret)
	subsonicService := subsonic.NewService(db.DB, authService, cfg.LastFMAPIKey)

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Test the star functionality
	fmt.Println("🧪 Testing Favorites (Star/Unstar) Functionality")
	fmt.Println("==================================================")

	// First, let's get a song ID from the database to test with
	songId := getSampleSongID(db.DB)
	if songId == "" {
		fmt.Println("❌ No songs found in database. Please run the scanner first.")
		os.Exit(1)
	}

	fmt.Printf("✅ Found sample song ID: %s\n", songId)

	// Test Star endpoint
	fmt.Println("\n🌟 Testing Star endpoint...")
	testStar(subsonicService, songId)

	// Test GetStarred endpoint
	fmt.Println("\n📋 Testing GetStarred endpoint...")
	testGetStarred(subsonicService)

	// Test Unstar endpoint
	fmt.Println("\n💫 Testing Unstar endpoint...")
	testUnstar(subsonicService, songId)

	// Test GetStarred again to verify removal
	fmt.Println("\n📋 Testing GetStarred after unstar...")
	testGetStarred(subsonicService)

	fmt.Println("\n✅ All tests completed!")
}

func getSampleSongID(db *sql.DB) string {
	var songId string
	err := db.QueryRow("SELECT id FROM songs LIMIT 1").Scan(&songId)
	if err != nil {
		return ""
	}
	return songId
}

func testStar(service *subsonic.Service, songId string) {
	router := gin.New()
	router.GET("/rest/star", service.Star)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/rest/star?id=%s", songId), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Printf("   Status: %d\n", w.Code)
	fmt.Printf("   Response: %s\n", w.Body.String())

	if w.Code == 200 {
		fmt.Println("   ✅ Star endpoint working")
	} else {
		fmt.Println("   ❌ Star endpoint failed")
	}
}

func testUnstar(service *subsonic.Service, songId string) {
	router := gin.New()
	router.GET("/rest/unstar", service.Unstar)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/rest/unstar?id=%s", songId), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Printf("   Status: %d\n", w.Code)
	fmt.Printf("   Response: %s\n", w.Body.String())

	if w.Code == 200 {
		fmt.Println("   ✅ Unstar endpoint working")
	} else {
		fmt.Println("   ❌ Unstar endpoint failed")
	}
}

func testGetStarred(service *subsonic.Service) {
	router := gin.New()
	router.GET("/rest/getStarred", service.GetStarred)

	req, _ := http.NewRequest("GET", "/rest/getStarred", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	fmt.Printf("   Status: %d\n", w.Code)
	fmt.Printf("   Response: %s\n", w.Body.String())

	if w.Code == 200 {
		fmt.Println("   ✅ GetStarred endpoint working")
	} else {
		fmt.Println("   ❌ GetStarred endpoint failed")
	}
}
