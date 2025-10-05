package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func Initialize(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run initial migrations if needed
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DB{db}, nil
}

func runMigrations(db *sql.DB) error {
	// Create users table
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		subscription_plan VARCHAR(50) DEFAULT 'free',
		max_concurrent_streams INTEGER DEFAULT 1,
		max_downloads_per_day INTEGER DEFAULT 10,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	)`

	if _, err := db.Exec(usersTable); err != nil {
		return err
	}

	// Create artists table
	artistsTable := `
	CREATE TABLE IF NOT EXISTS artists (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		bio TEXT,
		created_at TIMESTAMP DEFAULT NOW()
	)`

	if _, err := db.Exec(artistsTable); err != nil {
		return err
	}

	// Create albums table
	albumsTable := `
	CREATE TABLE IF NOT EXISTS albums (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		artist_id INTEGER REFERENCES artists(id),
		year INTEGER,
		genre VARCHAR(100),
		cover_art_path VARCHAR(500),
		created_at TIMESTAMP DEFAULT NOW()
	)`

	if _, err := db.Exec(albumsTable); err != nil {
		return err
	}

	// Create songs table
	songsTable := `
	CREATE TABLE IF NOT EXISTS songs (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		artist_id INTEGER REFERENCES artists(id),
		album_id INTEGER REFERENCES albums(id),
		track_number INTEGER,
		duration INTEGER, -- in seconds
		file_path VARCHAR(500) NOT NULL,
		file_size BIGINT,
		bitrate INTEGER,
		format VARCHAR(10),
		created_at TIMESTAMP DEFAULT NOW()
	)`

	if _, err := db.Exec(songsTable); err != nil {
		return err
	}

	// Create user sessions table for tracking concurrent streams
	sessionsTable := `
	CREATE TABLE IF NOT EXISTS user_sessions (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id),
		session_token VARCHAR(255) UNIQUE NOT NULL,
		ip_address INET,
		user_agent TEXT,
		created_at TIMESTAMP DEFAULT NOW(),
		last_activity TIMESTAMP DEFAULT NOW()
	)`

	if _, err := db.Exec(sessionsTable); err != nil {
		return err
	}

	// Create downloads table for tracking daily downloads
	downloadsTable := `
	CREATE TABLE IF NOT EXISTS downloads (
		id SERIAL PRIMARY KEY,
		user_id INTEGER REFERENCES users(id),
		song_id INTEGER REFERENCES songs(id),
		downloaded_at TIMESTAMP DEFAULT NOW()
	)`

	if _, err := db.Exec(downloadsTable); err != nil {
		return err
	}

	return nil
}
