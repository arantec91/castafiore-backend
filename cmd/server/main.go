package main

import (
	"fmt"
	"log"
	"net/http"

	"castafiore-backend/internal/api"
	"castafiore-backend/internal/auth"
	"castafiore-backend/internal/config"
	"castafiore-backend/internal/database"

	"github.com/gin-gonic/gin"
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

	// Setup router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Setup API routes
	api.SetupRoutes(router, authService, db.DB, cfg)

	// Start server
	address := cfg.Host + ":" + cfg.Port
	fmt.Printf("🎵 Castafiore Backend running on %s\n", address)
	fmt.Printf("📡 Subsonic API available at http://%s/rest/*\n", address)
	fmt.Printf("🌐 Web Admin Interface available at http://%s/admin\n", address)

	if cfg.Host == "0.0.0.0" {
		fmt.Printf("🌍 Server accessible from external network\n")
		fmt.Printf("   Local access: http://localhost:%s\n", cfg.Port)
		fmt.Printf("   Network access: http://[your-ip]:%s\n", cfg.Port)
	}

	log.Fatal(http.ListenAndServe(address, router))
}
