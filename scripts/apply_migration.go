package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://castafiore_user:castafiore_password@localhost/castafiore?sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	fmt.Println("Connected to database successfully!")
	fmt.Println()

	// Apply migration: Add subsonic_password column
	fmt.Println("Step 1: Adding subsonic_password column...")
	_, err = db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS subsonic_password VARCHAR(255);
	`)
	if err != nil {
		log.Fatalf("Error adding column: %v", err)
	}
	fmt.Println("✓ Column added successfully!")
	fmt.Println()

	// Update existing users with default password
	fmt.Println("Step 2: Setting default password for existing users...")
	result, err := db.Exec(`
		UPDATE users SET subsonic_password = 'changeme' WHERE subsonic_password IS NULL;
	`)
	if err != nil {
		log.Fatalf("Error updating users: %v", err)
	}
	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✓ Updated %d users with default password\n", rowsAffected)
	fmt.Println()

	// Update admin user with correct password
	fmt.Println("Step 3: Setting admin password...")
	_, err = db.Exec(`
		UPDATE users 
		SET subsonic_password = 'admin123' 
		WHERE username = 'admin' AND (subsonic_password IS NULL OR subsonic_password = 'changeme');
	`)
	if err != nil {
		log.Fatalf("Error updating admin: %v", err)
	}
	fmt.Println("✓ Admin password set to 'admin123'")
	fmt.Println()

	// Show current user status
	fmt.Println("Step 4: Current user status:")
	fmt.Println("----------------------------------------")
	rows, err := db.Query(`
		SELECT id, username, email, 
		       CASE 
		           WHEN subsonic_password IS NULL THEN 'NOT SET'
		           WHEN subsonic_password = 'changeme' THEN 'DEFAULT (needs update)'
		           ELSE 'CONFIGURED'
		       END as subsonic_status
		FROM users
		ORDER BY id;
	`)
	if err != nil {
		log.Fatalf("Error querying users: %v", err)
	}
	defer rows.Close()

	fmt.Printf("%-5s %-20s %-30s %-20s\n", "ID", "Username", "Email", "Subsonic Status")
	fmt.Println("----------------------------------------")
	for rows.Next() {
		var id int
		var username, email, status string
		if err := rows.Scan(&id, &username, &email, &status); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("%-5d %-20s %-30s %-20s\n", id, username, email, status)
	}
	fmt.Println("----------------------------------------")
	fmt.Println()

	fmt.Println("=== Migration Applied Successfully! ===")
	fmt.Println()
	fmt.Println("IMPORTANT: You need to set the subsonic_password for each user:")
	fmt.Println("  For user 'antonio', run this SQL:")
	fmt.Println("    UPDATE users SET subsonic_password = 'actual_password' WHERE username = 'antonio';")
	fmt.Println()
	fmt.Println("After setting passwords, restart the server.")
}
