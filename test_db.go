package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Connect to database
	db, err := sql.Open("postgres", "postgres://castafiore_user:castafiore_password@localhost/castafiore?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	fmt.Println("✓ Database connection successful")

	// Check artists count
	var artistCount int
	err = db.QueryRow("SELECT COUNT(*) FROM artists").Scan(&artistCount)
	if err != nil {
		log.Fatal("Failed to count artists:", err)
	}
	fmt.Printf("✓ Artists in database: %d\n", artistCount)

	// Check albums count
	var albumCount int
	err = db.QueryRow("SELECT COUNT(*) FROM albums").Scan(&albumCount)
	if err != nil {
		log.Fatal("Failed to count albums:", err)
	}
	fmt.Printf("✓ Albums in database: %d\n", albumCount)

	// Check songs count
	var songCount int
	err = db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&songCount)
	if err != nil {
		log.Fatal("Failed to count songs:", err)
	}
	fmt.Printf("✓ Songs in database: %d\n", songCount)

	// Test search for "arrolladora"
	fmt.Println("\n--- Testing search for 'arrolladora' ---")
	searchTerm := "%arrolladora%"

	rows, err := db.Query(`
		SELECT ar.id, ar.name, COUNT(al.id) as album_count
		FROM artists ar
		LEFT JOIN albums al ON ar.id = al.artist_id
		WHERE LOWER(ar.name) LIKE $1
		GROUP BY ar.id, ar.name
		ORDER BY ar.name
		LIMIT 10`, searchTerm)

	if err != nil {
		log.Fatal("Failed to search artists:", err)
	}
	defer rows.Close()

	artistsFound := 0
	for rows.Next() {
		var id, name string
		var albums int
		err := rows.Scan(&id, &name, &albums)
		if err != nil {
			log.Fatal("Failed to scan row:", err)
		}
		fmt.Printf("  Artist: %s (ID: %s, Albums: %d)\n", name, id, albums)
		artistsFound++
	}

	if artistsFound == 0 {
		fmt.Println("  ⚠ No artists found matching 'arrolladora'")
	} else {
		fmt.Printf("✓ Found %d artist(s)\n", artistsFound)
	}

	// Check user antonio
	fmt.Println("\n--- Checking user 'antonio' ---")
	var userID int
	var username, email string
	var isAdmin bool
	err = db.QueryRow("SELECT id, username, email, is_admin FROM users WHERE username = $1", "antonio").Scan(&userID, &username, &email, &isAdmin)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("  ⚠ User 'antonio' not found")
		} else {
			log.Fatal("Failed to query user:", err)
		}
	} else {
		fmt.Printf("✓ User found: %s (ID: %d, Email: %s, Admin: %v)\n", username, userID, email, isAdmin)
	}

	// List first 5 artists
	fmt.Println("\n--- First 5 artists in database ---")
	rows2, err := db.Query("SELECT id, name FROM artists ORDER BY name LIMIT 5")
	if err != nil {
		log.Fatal("Failed to query artists:", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var id, name string
		err := rows2.Scan(&id, &name)
		if err != nil {
			log.Fatal("Failed to scan artist:", err)
		}
		fmt.Printf("  %s: %s\n", id, name)
	}
}
