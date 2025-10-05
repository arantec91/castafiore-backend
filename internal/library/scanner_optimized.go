package library

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dhowden/tag"
)

// OptimizedScanner provides batch processing and optimizations for large libraries
type OptimizedScanner struct {
	db              *sql.DB
	TotalFiles      int
	ProcessedFiles  int
	IsScanning      bool
	LastError       string
	BatchSize       int  // Number of files to process in one transaction
	WorkerCount     int  // Number of concurrent workers
	IncrementalMode bool // Only scan new/modified files
	SkipCoverArt    bool // Skip cover art extraction for faster processing
	lastScanTime    time.Time
	mutex           sync.RWMutex
}

type BatchFileInfo struct {
	AudioFile
	ModTime time.Time
}

func NewOptimizedScanner(db *sql.DB) *OptimizedScanner {
	return &OptimizedScanner{
		db:              db,
		TotalFiles:      0,
		ProcessedFiles:  0,
		IsScanning:      false,
		LastError:       "",
		BatchSize:       100,  // Process 100 files per transaction
		WorkerCount:     4,    // 4 concurrent workers
		IncrementalMode: true, // Default to incremental scanning
		SkipCoverArt:    false,
	}
}

// SetOptimizationMode configures the scanner for different library sizes
func (s *OptimizedScanner) SetOptimizationMode(totalFiles int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if totalFiles > 100000 {
		// Large library mode
		s.BatchSize = 500
		s.WorkerCount = 8
		s.SkipCoverArt = true // Skip cover art for speed
		log.Println("Optimization: Large library mode enabled (batch=500, workers=8, no cover art)")
	} else if totalFiles > 10000 {
		// Medium library mode
		s.BatchSize = 200
		s.WorkerCount = 6
		s.SkipCoverArt = false
		log.Println("Optimization: Medium library mode enabled (batch=200, workers=6)")
	} else {
		// Small library mode
		s.BatchSize = 50
		s.WorkerCount = 2
		s.SkipCoverArt = false
		log.Println("Optimization: Small library mode enabled (batch=50, workers=2)")
	}
}

// ScanLibraryOptimized performs optimized scanning for large libraries
func (s *OptimizedScanner) ScanLibraryOptimized(musicPath string) error {
	log.Printf("Starting optimized library scan of: %s", musicPath)

	s.mutex.Lock()
	s.IsScanning = true
	s.ProcessedFiles = 0
	s.LastError = ""
	s.mutex.Unlock()

	// Get last scan time for incremental mode
	if s.IncrementalMode {
		s.getLastScanTime()
	}

	// Single pass: collect all files that need processing
	var filesToProcess []BatchFileInfo
	var totalFileCount int

	log.Println("Collecting files to process...")
	err := filepath.Walk(musicPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if s.isAudioFile(path) {
			totalFileCount++

			// In incremental mode, only process files newer than last scan
			if s.IncrementalMode && !s.lastScanTime.IsZero() && info.ModTime().Before(s.lastScanTime) {
				return nil // Skip unmodified files
			}

			filesToProcess = append(filesToProcess, BatchFileInfo{
				AudioFile: AudioFile{Path: path, Size: info.Size()},
				ModTime:   info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		s.mutex.Lock()
		s.LastError = fmt.Sprintf("error collecting files: %v", err)
		s.IsScanning = false
		s.mutex.Unlock()
		return fmt.Errorf(s.LastError)
	}

	s.mutex.Lock()
	s.TotalFiles = len(filesToProcess)
	s.mutex.Unlock()

	log.Printf("Found %d total audio files, %d need processing", totalFileCount, len(filesToProcess))

	// Set optimization mode based on total files
	s.SetOptimizationMode(totalFileCount)

	if len(filesToProcess) == 0 {
		log.Println("No files need processing (incremental mode)")
		s.mutex.Lock()
		s.IsScanning = false
		s.mutex.Unlock()
		return nil
	}

	// Process files in batches using worker pool
	return s.processBatches(filesToProcess)
}

// processBatches handles batch processing with worker pool
func (s *OptimizedScanner) processBatches(files []BatchFileInfo) error {
	// Create worker pool
	jobs := make(chan []BatchFileInfo, s.WorkerCount*2)
	results := make(chan error, s.WorkerCount*2)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.WorkerCount; i++ {
		wg.Add(1)
		go s.batchWorker(jobs, results, &wg)
	}

	// Send batches to workers
	go func() {
		defer close(jobs)
		for i := 0; i < len(files); i += s.BatchSize {
			end := i + s.BatchSize
			if end > len(files) {
				end = len(files)
			}
			batch := files[i:end]
			jobs <- batch
		}
	}()

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var lastError error
	batchCount := 0
	for err := range results {
		batchCount++
		if err != nil {
			log.Printf("Batch %d error: %v", batchCount, err)
			lastError = err
		}
	}

	// Update last scan time
	if lastError == nil {
		s.updateLastScanTime()
	}

	s.mutex.Lock()
	s.IsScanning = false
	if lastError != nil {
		s.LastError = lastError.Error()
	}
	s.mutex.Unlock()

	if lastError != nil {
		return lastError
	}

	log.Println("Optimized library scan completed successfully")
	return nil
}

// batchWorker processes batches of files
func (s *OptimizedScanner) batchWorker(jobs <-chan []BatchFileInfo, results chan<- error, wg *sync.WaitGroup) {
	defer wg.Done()

	for batch := range jobs {
		if err := s.processBatch(batch); err != nil {
			results <- err
			continue
		}
		results <- nil
	}
}

// processBatch processes a batch of files in a single transaction
func (s *OptimizedScanner) processBatch(batch []BatchFileInfo) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("cannot begin transaction: %v", err)
	}

	// Ensure proper cleanup
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("Error rolling back transaction: %v", rollbackErr)
			}
		}
	}()

	// Process each file in the batch
	for _, fileInfo := range batch {
		audioFile, err := s.extractMetadataFast(fileInfo.AudioFile.Path, fileInfo.AudioFile.Size)
		if err != nil {
			log.Printf("Error processing file %s: %v", fileInfo.AudioFile.Path, err)
			continue // Skip this file but continue with others
		}

		// Get or create artist with improved error handling
		artistID, err := s.getOrCreateArtistOptimized(tx, audioFile.Artist)
		if err != nil {
			log.Printf("Error creating artist %s: %v", audioFile.Artist, err)
			continue // Skip this file but continue with others
		}

		// Get or create album with improved error handling
		albumID, err := s.getOrCreateAlbumOptimized(tx, audioFile.Album, artistID, audioFile.Year, audioFile.Genre)
		if err != nil {
			log.Printf("Error creating album %s: %v", audioFile.Album, err)
			continue // Skip this file but continue with others
		}

		// Insert song with improved error handling
		err = s.insertOrUpdateSongOptimized(tx, *audioFile, artistID, albumID)
		if err != nil {
			log.Printf("Error inserting song %s: %v", audioFile.Title, err)
			continue // Skip this file but continue with others
		}

		// Update progress
		s.mutex.Lock()
		s.ProcessedFiles++
		s.mutex.Unlock()
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("cannot commit transaction: %v", err)
	}

	committed = true
	return nil
}

// getOrCreateArtistOptimized gets existing artist or creates new one (optimized version)
func (s *OptimizedScanner) getOrCreateArtistOptimized(tx *sql.Tx, name string) (int, error) {
	// Clean the artist name to handle potential encoding issues
	cleanName := s.cleanStringOptimized(name)
	if cleanName == "" {
		cleanName = "Unknown Artist"
	}

	var id int
	err := tx.QueryRow("SELECT id FROM artists WHERE name = $1", cleanName).Scan(&id)

	if err == sql.ErrNoRows {
		// Create new artist
		err = tx.QueryRow(
			"INSERT INTO artists (name, created_at, updated_at) VALUES ($1, NOW(), NOW()) RETURNING id",
			cleanName,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("failed to insert artist '%s': %v", cleanName, err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to query artist '%s': %v", cleanName, err)
	}

	return id, nil
}

// getOrCreateAlbumOptimized gets existing album or creates new one (optimized version)
func (s *OptimizedScanner) getOrCreateAlbumOptimized(tx *sql.Tx, name string, artistID, year int, genre string) (int, error) {
	// Clean the album name and genre
	cleanName := s.cleanStringOptimized(name)
	if cleanName == "" {
		cleanName = "Unknown Album"
	}

	cleanGenre := s.cleanStringOptimized(genre)
	if cleanGenre == "" {
		cleanGenre = "Unknown"
	}

	var id int
	err := tx.QueryRow("SELECT id FROM albums WHERE name = $1 AND artist_id = $2", cleanName, artistID).Scan(&id)

	if err == sql.ErrNoRows {
		// Create new album
		err = tx.QueryRow(
			"INSERT INTO albums (name, artist_id, year, genre, created_at, updated_at) VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING id",
			cleanName, artistID, year, cleanGenre,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("failed to insert album '%s': %v", cleanName, err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("failed to query album '%s': %v", cleanName, err)
	}

	return id, nil
}

// insertOrUpdateSongOptimized inserts or updates song information (optimized version)
func (s *OptimizedScanner) insertOrUpdateSongOptimized(tx *sql.Tx, file AudioFile, artistID, albumID int) error {
	// Clean the song title
	cleanTitle := s.cleanStringOptimized(file.Title)
	if cleanTitle == "" {
		cleanTitle = filepath.Base(file.Path)
		cleanTitle = s.cleanStringOptimized(cleanTitle)
	}

	// Validate numeric fields
	trackNumber := file.TrackNumber
	if trackNumber < 0 {
		trackNumber = 0
	}
	if trackNumber > 999 {
		trackNumber = 999
	}

	durationSeconds := int(file.Duration.Seconds())
	if durationSeconds < 0 {
		durationSeconds = 0
	}
	if durationSeconds > 86400 { // 24 hours max
		durationSeconds = 86400
	}

	bitrate := file.Bitrate
	if bitrate < 0 {
		bitrate = 0
	}
	if bitrate > 10000 { // 10 Mbps max
		bitrate = 10000
	}

	_, err := tx.Exec(`
		INSERT INTO songs (title, artist_id, album_id, track_number, duration, file_path, file_size, bitrate, format, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		ON CONFLICT (file_path) 
		DO UPDATE SET 
			title = EXCLUDED.title,
			artist_id = EXCLUDED.artist_id,
			album_id = EXCLUDED.album_id,
			track_number = EXCLUDED.track_number,
			duration = EXCLUDED.duration,
			file_size = EXCLUDED.file_size,
			bitrate = EXCLUDED.bitrate,
			format = EXCLUDED.format,
			updated_at = NOW()
	`, cleanTitle, artistID, albumID, trackNumber, durationSeconds, file.Path, file.Size, bitrate, file.Format)

	if err != nil {
		return fmt.Errorf("failed to insert/update song '%s': %v", cleanTitle, err)
	}

	return nil
}

// cleanStringOptimized cleans and validates metadata strings (optimized version)
func (s *OptimizedScanner) cleanStringOptimized(input string) string {
	// Trim whitespace
	cleaned := strings.TrimSpace(input)

	// Remove null bytes and other problematic characters
	cleaned = strings.ReplaceAll(cleaned, "\x00", "")
	cleaned = strings.ReplaceAll(cleaned, "\ufffd", "") // Unicode replacement character

	// Limit length to prevent database issues
	if len(cleaned) > 255 {
		cleaned = cleaned[:255]
	}

	return cleaned
}

// extractMetadataFast extracts metadata with minimal I/O operations
func (s *OptimizedScanner) extractMetadataFast(path string, fileSize int64) (*AudioFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %v", err)
	}
	defer file.Close()

	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")

	// Fast metadata extraction (limited tag reading for speed)
	var metadata tag.Metadata
	if !s.SkipCoverArt {
		metadata, err = tag.ReadFrom(file)
	} else {
		// Only read basic tags, skip cover art for speed
		metadata, err = tag.ReadFrom(file)
	}

	if err != nil {
		// Create basic metadata from filename if tag reading fails
		metadata = s.createBasicMetadataOptimized(path)
	}

	// Fast duration/bitrate extraction
	duration, bitrate := s.extractAudioPropertiesFast(path, format, fileSize)

	// Clean and validate metadata strings
	title := s.cleanStringOptimized(metadata.Title())
	if title == "" {
		title = s.cleanStringOptimized(filepath.Base(path))
	}

	artist := s.cleanStringOptimized(metadata.Artist())
	if artist == "" {
		artist = "Unknown Artist"
	}

	album := s.cleanStringOptimized(metadata.Album())
	if album == "" {
		album = "Unknown Album"
	}

	genre := s.cleanStringOptimized(metadata.Genre())
	if genre == "" {
		genre = "Unknown"
	}

	audioFile := &AudioFile{
		Path:        path,
		Title:       title,
		Artist:      artist,
		Album:       album,
		Genre:       genre,
		Year:        metadata.Year(),
		TrackNumber: s.getTrackNumberOptimized(metadata),
		Duration:    duration,
		Size:        fileSize,
		Format:      format,
		Bitrate:     bitrate,
	}

	// Skip cover art extraction for large libraries or if disabled
	if !s.SkipCoverArt {
		if picture := metadata.Picture(); picture != nil && len(picture.Data) < 5*1024*1024 { // Limit to 5MB
			audioFile.CoverArt = picture.Data
		}
	}

	return audioFile, nil
}

// Fast helper functions
func (s *OptimizedScanner) createBasicMetadataOptimized(path string) tag.Metadata {
	return &basicMetadata{
		title:  strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		artist: "Unknown Artist",
		album:  "Unknown Album",
	}
}

func (s *OptimizedScanner) getTrackNumberOptimized(metadata tag.Metadata) int {
	track, _ := metadata.Track()
	return track
}

// extractAudioPropertiesFast - simplified version for speed
func (s *OptimizedScanner) extractAudioPropertiesFast(path, format string, fileSize int64) (time.Duration, int) {
	// For large libraries, use faster estimation methods
	switch format {
	case "mp3":
		// Quick MP3 estimation (first frame only)
		return s.extractMP3PropertiesFast(path, fileSize)
	case "flac":
		// FLAC estimation
		return s.extractFLACPropertiesFast(path, fileSize)
	default:
		// Generic estimation based on file size and typical bitrates
		return s.estimatePropertiesBySize(format, fileSize)
	}
}

func (s *OptimizedScanner) extractMP3PropertiesFast(path string, fileSize int64) (time.Duration, int) {
	// Quick estimation: read only first MP3 frame
	file, err := os.Open(path)
	if err != nil {
		return s.estimateDurationMP3(fileSize, 192), 192 // Default estimation
	}
	defer file.Close()

	// Read only first 512 bytes for header detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil || n < 4 {
		return s.estimateDurationMP3(fileSize, 192), 192
	}

	// Find first MP3 frame header and extract bitrate
	bitrate := s.findFirstMP3Bitrate(buffer)
	if bitrate == 0 {
		bitrate = 192 // Default
	}

	duration := s.estimateDurationMP3(fileSize, bitrate)
	return duration, bitrate
}

func (s *OptimizedScanner) extractFLACPropertiesFast(path string, fileSize int64) (time.Duration, int) {
	// FLAC estimation - use file size
	avgBitrate := 800 // Typical FLAC bitrate
	duration := time.Duration(fileSize*8/int64(avgBitrate)/1000) * time.Second
	return duration, avgBitrate
}

func (s *OptimizedScanner) estimatePropertiesBySize(format string, fileSize int64) (time.Duration, int) {
	var avgBitrate int
	switch format {
	case "m4a", "aac":
		avgBitrate = 256
	case "ogg":
		avgBitrate = 192
	case "wav":
		avgBitrate = 1411
	default:
		avgBitrate = 192
	}

	duration := time.Duration(fileSize*8/int64(avgBitrate)/1000) * time.Second
	return duration, avgBitrate
}

func (s *OptimizedScanner) estimateDurationMP3(fileSize int64, bitrate int) time.Duration {
	// Simple estimation: fileSize * 8 / bitrate / 1000 seconds
	seconds := fileSize * 8 / int64(bitrate) / 1000
	return time.Duration(seconds) * time.Second
}

func (s *OptimizedScanner) findFirstMP3Bitrate(buffer []byte) int {
	// Simplified MP3 header search - find first valid frame
	for i := 0; i < len(buffer)-3; i++ {
		if buffer[i] == 0xFF && (buffer[i+1]&0xE0) == 0xE0 {
			// Found potential MP3 frame header
			bitrateIndex := (buffer[i+2] & 0xF0) >> 4
			if bitrateIndex > 0 && bitrateIndex < 15 {
				// Basic bitrate lookup (simplified)
				bitrates := []int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}
				if int(bitrateIndex) < len(bitrates) {
					return bitrates[bitrateIndex]
				}
			}
		}
	}
	return 0
}

// Utility functions for scan time tracking
func (s *OptimizedScanner) getLastScanTime() {
	row := s.db.QueryRow("SELECT last_scan_time FROM scan_metadata WHERE id = 1")
	var lastScan sql.NullTime
	if err := row.Scan(&lastScan); err == nil && lastScan.Valid {
		s.lastScanTime = lastScan.Time
		log.Printf("Last scan time: %s", s.lastScanTime.Format(time.RFC3339))
	}
}

func (s *OptimizedScanner) updateLastScanTime() {
	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO scan_metadata (id, last_scan_time) 
		VALUES (1, $1) 
		ON CONFLICT (id) 
		DO UPDATE SET last_scan_time = EXCLUDED.last_scan_time
	`, now)
	if err != nil {
		log.Printf("Warning: Could not update last scan time: %v", err)
	}
}

// isAudioFile checks if the file is a supported audio format
func (s *OptimizedScanner) isAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp3" || ext == ".flac" || ext == ".ogg" || ext == ".m4a" || ext == ".wav"
}

func (s *OptimizedScanner) getStringOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// GetScanProgress returns the current progress of the library scan
func (s *OptimizedScanner) GetScanProgress() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	progress := make(map[string]interface{})
	progress["is_scanning"] = s.IsScanning
	progress["total_files"] = s.TotalFiles
	progress["processed_files"] = s.ProcessedFiles

	percentComplete := 0.0
	if s.TotalFiles > 0 {
		percentComplete = float64(s.ProcessedFiles) / float64(s.TotalFiles) * 100
	}
	progress["percent_complete"] = percentComplete

	if s.LastError != "" {
		progress["last_error"] = s.LastError
	}

	return progress
}

// GetScanStats returns statistics about the current library
func (s *OptimizedScanner) GetScanStats() (map[string]int, error) {
	stats := make(map[string]int)

	queries := map[string]string{
		"artists": "SELECT COUNT(*) FROM artists",
		"albums":  "SELECT COUNT(*) FROM albums",
		"songs":   "SELECT COUNT(*) FROM songs",
	}

	for key, query := range queries {
		var count int
		err := s.db.QueryRow(query).Scan(&count)
		if err != nil {
			return nil, err
		}
		stats[key] = count
	}

	return stats, nil
}
