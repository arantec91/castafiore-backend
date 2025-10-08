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

	// Update antonio's password
	fmt.Println("Updating password for user 'antonio'...")
	_, err = db.Exec(`
		UPDATE users SET subsonic_password = $1 WHERE username = $2;
	`, "150291", "antonio")
	if err != nil {
		log.Fatalf("Error updating antonio: %v", err)
	}
	fmt.Println("✓ Password updated for antonio")

	// Update fredyaran's password
	fmt.Println("Updating password for user 'fredyaran'...")
	_, err = db.Exec(`
		UPDATE users SET subsonic_password = $1 WHERE username = $2;
	`, "Aleida2001+", "fredyaran")
	if err != nil {
		log.Fatalf("Error updating fredyaran: %v", err)
	}
	fmt.Println("✓ Password updated for fredyaran")
	fmt.Println()

	// Show current user status
	fmt.Println("Current user status:")
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

	fmt.Println("=== All passwords updated successfully! ===")
	fmt.Println()
	fmt.Println("You can now restart the server and test Subsonic authentication.")
}
