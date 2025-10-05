package library

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

type Scanner struct {
	db *sql.DB
	// Progress tracking
	TotalFiles     int
	ProcessedFiles int
	IsScanning     bool
	LastError      string
}

type AudioFile struct {
	Path        string
	Title       string
	Artist      string
	Album       string
	Genre       string
	Year        int
	TrackNumber int
	Duration    time.Duration
	Size        int64
	Format      string
	Bitrate     int
	CoverArt    []byte // Cover art image data
}

func NewScanner(db *sql.DB) *Scanner {
	return &Scanner{
		db:             db,
		TotalFiles:     0,
		ProcessedFiles: 0,
		IsScanning:     false,
		LastError:      "",
	}
}

// ScanLibrary scans the music directory and updates the database
func (s *Scanner) ScanLibrary(musicPath string) error {
	log.Printf("Starting library scan of: %s", musicPath)

	// Reset progress tracking
	s.TotalFiles = 0
	s.ProcessedFiles = 0
	s.IsScanning = true
	s.LastError = ""

	// First pass: count total audio files
	log.Println("Counting audio files...")
	err := filepath.Walk(musicPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue scanning
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Count only audio files
		if s.isAudioFile(path) {
			s.TotalFiles++
		}

		return nil
	})

	if err != nil {
		s.LastError = fmt.Sprintf("error counting files: %v", err)
		s.IsScanning = false
		return fmt.Errorf(s.LastError)
	}

	log.Printf("Found %d audio files to process", s.TotalFiles)

	// Clear existing data (optional - comment out if you want to keep existing data)
	if err := s.clearExistingData(); err != nil {
		log.Printf("Warning: Could not clear existing data: %v", err)
	}

	// Second pass: process files and update progress
	log.Println("Processing audio files...")
	err = filepath.Walk(musicPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return nil // Continue scanning
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if it's an audio file
		if s.isAudioFile(path) {
			if err := s.processAudioFile(path, info); err != nil {
				log.Printf("Error processing file %s: %v", path, err)
			}
			s.ProcessedFiles++

			// Log progress every 100 files
			if s.ProcessedFiles%100 == 0 || s.ProcessedFiles == s.TotalFiles {
				progress := float64(s.ProcessedFiles) / float64(s.TotalFiles) * 100
				log.Printf("Progress: %.2f%% (%d/%d files processed)", progress, s.ProcessedFiles, s.TotalFiles)
			}
		}

		return nil
	})

	if err != nil {
		s.LastError = fmt.Sprintf("error scanning library: %v", err)
		s.IsScanning = false
		return fmt.Errorf(s.LastError)
	}

	s.IsScanning = false
	log.Println("Library scan completed")
	return nil
}

// isAudioFile checks if the file is a supported audio format
func (s *Scanner) isAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	supportedFormats := []string{".mp3", ".flac", ".m4a", ".ogg", ".wav"}

	for _, format := range supportedFormats {
		if ext == format {
			return true
		}
	}
	return false
}

// processAudioFile extracts metadata and adds to database
func (s *Scanner) processAudioFile(path string, info os.FileInfo) error {
	// Open the file for tag reading
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open file: %v", err)
	}
	defer file.Close()

	// Extract metadata
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		log.Printf("Warning: Could not read tags from %s: %v", path, err)
		// Create basic metadata from filename if tag reading fails
		metadata = s.createBasicMetadata(path)
	}

	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")

	// Extract duration and bitrate
	duration, bitrate := s.extractAudioProperties(path, format, info.Size())

	// Extract cover art (with size limit to prevent memory issues)
	var coverArt []byte
	if picture := metadata.Picture(); picture != nil && len(picture.Data) < 5*1024*1024 { // Limit to 5MB
		coverArt = picture.Data
	}

	// Clean and validate metadata strings
	title := s.cleanString(metadata.Title())
	if title == "" {
		title = s.cleanString(filepath.Base(path))
	}

	artist := s.cleanString(metadata.Artist())
	if artist == "" {
		artist = "Unknown Artist"
	}

	album := s.cleanString(metadata.Album())
	if album == "" {
		album = "Unknown Album"
	}

	genre := s.cleanString(metadata.Genre())
	if genre == "" {
		genre = "Unknown"
	}

	audioFile := AudioFile{
		Path:        path,
		Title:       title,
		Artist:      artist,
		Album:       album,
		Genre:       genre,
		Year:        metadata.Year(),
		TrackNumber: s.getTrackNumber(metadata),
		Duration:    duration,
		Size:        info.Size(),
		Format:      format,
		Bitrate:     bitrate,
		CoverArt:    coverArt,
	}

	// Add to database
	return s.addToDatabase(audioFile)
}

// cleanString cleans and validates metadata strings
func (s *Scanner) cleanString(input string) string {
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

// createBasicMetadata creates metadata from filename when tag reading fails
func (s *Scanner) createBasicMetadata(path string) tag.Metadata {
	// This is a simple implementation - you might want to enhance it
	// to parse more information from file/folder structure
	return &basicMetadata{
		title:  filepath.Base(path),
		artist: "Unknown Artist",
		album:  "Unknown Album",
	}
}

// basicMetadata implements tag.Metadata interface for fallback cases
type basicMetadata struct {
	title, artist, album string
}

func (m *basicMetadata) Format() tag.Format          { return tag.UnknownFormat }
func (m *basicMetadata) FileType() tag.FileType      { return tag.UnknownFileType }
func (m *basicMetadata) Title() string               { return m.title }
func (m *basicMetadata) Album() string               { return m.album }
func (m *basicMetadata) Artist() string              { return m.artist }
func (m *basicMetadata) AlbumArtist() string         { return m.artist }
func (m *basicMetadata) Composer() string            { return "" }
func (m *basicMetadata) Genre() string               { return "Unknown" }
func (m *basicMetadata) Year() int                   { return 0 }
func (m *basicMetadata) Track() (int, int)           { return 0, 0 }
func (m *basicMetadata) Disc() (int, int)            { return 0, 0 }
func (m *basicMetadata) Picture() *tag.Picture       { return nil }
func (m *basicMetadata) Lyrics() string              { return "" }
func (m *basicMetadata) Comment() string             { return "" }
func (m *basicMetadata) Raw() map[string]interface{} { return nil }

// getTrackNumber extracts track number from metadata
func (s *Scanner) getTrackNumber(metadata tag.Metadata) int {
	track, _ := metadata.Track()
	return track
}

// extractAudioProperties extracts duration and bitrate from audio files
func (s *Scanner) extractAudioProperties(path string, format string, fileSize int64) (time.Duration, int) {
	var duration time.Duration
	var bitrate int

	switch format {
	case "mp3":
		duration, bitrate = s.extractMP3Properties(path, fileSize)
	case "flac":
		duration, bitrate = s.extractFLACProperties(path, fileSize)
	case "m4a":
		duration, bitrate = s.extractM4AProperties(path, fileSize)
	case "ogg":
		duration, bitrate = s.extractOGGProperties(path, fileSize)
	case "wav":
		duration, bitrate = s.extractWAVProperties(path, fileSize)
	default:
		// Fallback: estimate based on file size (assuming 128 kbps)
		bitrate = 128
		duration = time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
	}

	return duration, bitrate
}

// extractMP3Properties extracts duration and bitrate from MP3 files
func (s *Scanner) extractMP3Properties(path string, fileSize int64) (time.Duration, int) {
	file, err := os.Open(path)
	if err != nil {
		// Fallback: estimate based on file size (assuming 128 kbps)
		bitrate := 128
		duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
		return duration, bitrate
	}
	defer file.Close()

	// MP3 frame header parsing
	var totalFrames int64
	var totalBitrate int64
	var sampleRate int

	buf := make([]byte, 4)
	for {
		_, err := file.Read(buf)
		if err != nil {
			break
		}

		// Check for MP3 frame sync (11 bits set)
		if buf[0] == 0xFF && (buf[1]&0xE0) == 0xE0 {
			// Parse MP3 frame header
			version := (buf[1] >> 3) & 0x03
			layer := (buf[1] >> 1) & 0x03
			bitrateIndex := (buf[2] >> 4) & 0x0F
			sampleRateIndex := (buf[2] >> 2) & 0x03

			// Get bitrate (kbps)
			bitrate := s.getMP3Bitrate(version, layer, bitrateIndex)
			if bitrate == 0 {
				continue
			}

			// Get sample rate
			sampleRate = s.getMP3SampleRate(version, sampleRateIndex)
			if sampleRate == 0 {
				continue
			}

			totalFrames++
			totalBitrate += int64(bitrate)

			// Calculate frame size and skip to next frame
			padding := (buf[2] >> 1) & 0x01
			frameSize := (144*bitrate*1000)/sampleRate + int(padding)
			if frameSize > 4 {
				file.Seek(int64(frameSize-4), io.SeekCurrent)
			}

			// Sample first 100 frames for performance
			if totalFrames >= 100 {
				break
			}
		}
	}

	if totalFrames > 0 && sampleRate > 0 {
		avgBitrate := int(totalBitrate / totalFrames)

		// Calculate duration based on file size and average bitrate
		if avgBitrate > 0 {
			durationSeconds := float64(fileSize*8) / float64(avgBitrate*1000)
			duration := time.Duration(durationSeconds * float64(time.Second))
			return duration, avgBitrate
		}
	}

	// Fallback: estimate based on file size (assuming 128 kbps)
	bitrate := 128
	duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
	return duration, bitrate
}

// getMP3Bitrate returns the bitrate for MP3 frame
func (s *Scanner) getMP3Bitrate(version, layer, bitrateIndex byte) int {
	// Bitrate table for MPEG 1 Layer III
	bitrateTableV1L3 := []int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
	// Bitrate table for MPEG 2/2.5 Layer III
	bitrateTableV2L3 := []int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}

	if bitrateIndex >= 15 {
		return 0
	}

	if version == 3 { // MPEG 1
		return bitrateTableV1L3[bitrateIndex]
	}
	// MPEG 2 or 2.5
	return bitrateTableV2L3[bitrateIndex]
}

// getMP3SampleRate returns the sample rate for MP3 frame
func (s *Scanner) getMP3SampleRate(version, sampleRateIndex byte) int {
	sampleRates := map[byte][]int{
		3: {44100, 48000, 32000, 0}, // MPEG 1
		2: {22050, 24000, 16000, 0}, // MPEG 2
		0: {11025, 12000, 8000, 0},  // MPEG 2.5
	}

	if rates, ok := sampleRates[version]; ok && sampleRateIndex < 3 {
		return rates[sampleRateIndex]
	}
	return 0
}

// extractFLACProperties extracts duration and bitrate from FLAC files
func (s *Scanner) extractFLACProperties(path string, fileSize int64) (time.Duration, int) {
	file, err := os.Open(path)
	if err != nil {
		// Fallback
		bitrate := 1000 // FLAC typically ~1000 kbps
		duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
		return duration, bitrate
	}
	defer file.Close()

	// Read FLAC header
	header := make([]byte, 4)
	if _, err := file.Read(header); err != nil || string(header) != "fLaC" {
		// Fallback
		bitrate := 1000 // FLAC typically ~1000 kbps
		duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
		return duration, bitrate
	}

	// Read STREAMINFO block
	for {
		blockHeader := make([]byte, 4)
		if _, err := file.Read(blockHeader); err != nil {
			break
		}

		blockType := blockHeader[0] & 0x7F
		blockSize := int(blockHeader[1])<<16 | int(blockHeader[2])<<8 | int(blockHeader[3])

		if blockType == 0 { // STREAMINFO
			streamInfo := make([]byte, blockSize)
			if _, err := file.Read(streamInfo); err != nil {
				break
			}

			// Parse STREAMINFO
			sampleRate := (int(streamInfo[10]) << 12) | (int(streamInfo[11]) << 4) | (int(streamInfo[12]) >> 4)
			totalSamples := (int64(streamInfo[13]&0x0F) << 32) | (int64(streamInfo[14]) << 24) | (int64(streamInfo[15]) << 16) | (int64(streamInfo[16]) << 8) | int64(streamInfo[17])

			if sampleRate > 0 && totalSamples > 0 {
				duration := time.Duration(float64(totalSamples) / float64(sampleRate) * float64(time.Second))
				bitrate := int(float64(fileSize*8) / duration.Seconds() / 1000)
				return duration, bitrate
			}
			break
		}

		// Skip to next block
		file.Seek(int64(blockSize), io.SeekCurrent)

		// Check if this was the last block
		if blockHeader[0]&0x80 != 0 {
			break
		}
	}

	// Fallback
	bitrate := 1000
	duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
	return duration, bitrate
}

// extractM4AProperties extracts duration and bitrate from M4A files
func (s *Scanner) extractM4AProperties(path string, fileSize int64) (time.Duration, int) {
	// M4A parsing is complex, use estimation
	// Typical M4A bitrate is 256 kbps for AAC
	bitrate := 256
	duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
	return duration, bitrate
}

// extractOGGProperties extracts duration and bitrate from OGG files
func (s *Scanner) extractOGGProperties(path string, fileSize int64) (time.Duration, int) {
	file, err := os.Open(path)
	if err != nil {
		// Fallback
		bitrate := 192
		duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
		return duration, bitrate
	}
	defer file.Close()

	// Read OGG header
	header := make([]byte, 27)
	if _, err := file.Read(header); err != nil || string(header[0:4]) != "OggS" {
		// Fallback
		bitrate := 192
		duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
		return duration, bitrate
	}

	// Seek to end to find last granule position
	file.Seek(-65536, io.SeekEnd) // Read last 64KB
	buf := make([]byte, 65536)
	n, _ := file.Read(buf)

	var lastGranule int64
	// Search for last OggS page
	for i := n - 27; i >= 0; i-- {
		if string(buf[i:i+4]) == "OggS" {
			lastGranule = int64(binary.LittleEndian.Uint64(buf[i+6 : i+14]))
			break
		}
	}

	if lastGranule > 0 {
		// Assume 48000 Hz sample rate (common for Vorbis)
		sampleRate := 48000
		duration := time.Duration(float64(lastGranule) / float64(sampleRate) * float64(time.Second))
		bitrate := int(float64(fileSize*8) / duration.Seconds() / 1000)
		return duration, bitrate
	}

	// Fallback
	bitrate := 192
	duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
	return duration, bitrate
}

// extractWAVProperties extracts duration and bitrate from WAV files
func (s *Scanner) extractWAVProperties(path string, fileSize int64) (time.Duration, int) {
	file, err := os.Open(path)
	if err != nil {
		// Fallback
		bitrate := 1411 // CD quality: 44.1kHz * 16bit * 2 channels
		duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
		return duration, bitrate
	}
	defer file.Close()

	// Read RIFF header
	header := make([]byte, 12)
	if _, err := file.Read(header); err != nil || string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		// Fallback
		bitrate := 1411 // CD quality: 44.1kHz * 16bit * 2 channels
		duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
		return duration, bitrate
	}

	// Find fmt chunk
	for {
		chunkHeader := make([]byte, 8)
		if _, err := file.Read(chunkHeader); err != nil {
			break
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "fmt " {
			fmtData := make([]byte, chunkSize)
			if _, err := file.Read(fmtData); err != nil {
				break
			}

			// Parse fmt chunk
			sampleRate := binary.LittleEndian.Uint32(fmtData[4:8])
			byteRate := binary.LittleEndian.Uint32(fmtData[8:12])

			if byteRate > 0 {
				duration := time.Duration(float64(fileSize-44) / float64(byteRate) * float64(time.Second))
				bitrate := int(byteRate * 8 / 1000)
				return duration, bitrate
			}

			if sampleRate > 0 {
				// Estimate based on sample rate (assume 16-bit stereo)
				bitrate := int(sampleRate * 16 * 2 / 1000)
				duration := time.Duration(float64(fileSize-44) * 8 / float64(bitrate*1000) * float64(time.Second))
				return duration, bitrate
			}
			break
		}

		// Skip chunk data
		file.Seek(int64(chunkSize), io.SeekCurrent)
	}

	// Fallback
	bitrate := 1411
	duration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second
	return duration, bitrate
}

// addToDatabase adds the audio file information to the database
func (s *Scanner) addToDatabase(file AudioFile) error {
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

	// Get or create artist
	artistID, err := s.getOrCreateArtist(tx, file.Artist)
	if err != nil {
		return fmt.Errorf("cannot get/create artist %s: %v", file.Artist, err)
	}

	// Get or create album
	albumID, err := s.getOrCreateAlbum(tx, file.Album, artistID, file.Year, file.Genre, file.CoverArt)
	if err != nil {
		return fmt.Errorf("cannot get/create album %s: %v", file.Album, err)
	}

	// Insert song (or update if exists)
	err = s.insertOrUpdateSong(tx, file, artistID, albumID)
	if err != nil {
		return fmt.Errorf("cannot insert/update song %s: %v", file.Title, err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("cannot commit transaction: %v", err)
	}

	committed = true
	return nil
}

// getOrCreateArtist gets existing artist or creates new one
func (s *Scanner) getOrCreateArtist(tx *sql.Tx, name string) (int, error) {
	// Clean the artist name to handle potential encoding issues
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = "Unknown Artist"
	}

	// Limit name length to prevent database errors
	if len(cleanName) > 255 {
		cleanName = cleanName[:255]
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
		log.Printf("Created new artist: %s (ID: %d)", cleanName, id)
	} else if err != nil {
		return 0, fmt.Errorf("failed to query artist '%s': %v", cleanName, err)
	}

	return id, nil
}

// getOrCreateAlbum gets existing album or creates new one
func (s *Scanner) getOrCreateAlbum(tx *sql.Tx, name string, artistID, year int, genre string, coverArt []byte) (int, error) {
	// Clean the album name to handle potential encoding issues
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = "Unknown Album"
	}

	// Limit name length to prevent database errors
	if len(cleanName) > 255 {
		cleanName = cleanName[:255]
	}

	// Clean genre name
	cleanGenre := strings.TrimSpace(genre)
	if cleanGenre == "" {
		cleanGenre = "Unknown"
	}
	if len(cleanGenre) > 100 {
		cleanGenre = cleanGenre[:100]
	}

	var id int
	var existingCoverPath sql.NullString
	err := tx.QueryRow("SELECT id, cover_art_path FROM albums WHERE name = $1 AND artist_id = $2", cleanName, artistID).Scan(&id, &existingCoverPath)

	if err == sql.ErrNoRows {
		// Create new album with cover art
		coverArtPath := ""
		if len(coverArt) > 0 && !s.skipCoverArt() {
			coverArtPath = s.saveCoverArt(cleanName, artistID, coverArt)
		}

		err = tx.QueryRow(
			"INSERT INTO albums (name, artist_id, year, genre, cover_art_path, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id",
			cleanName, artistID, year, cleanGenre, coverArtPath,
		).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("failed to insert album '%s': %v", cleanName, err)
		}
		log.Printf("Created new album: %s (ID: %d)", cleanName, id)
	} else if err != nil {
		return 0, fmt.Errorf("failed to query album '%s': %v", cleanName, err)
	} else {
		// Album exists, update cover art if we have new one and there's no existing cover
		if len(coverArt) > 0 && !existingCoverPath.Valid && !s.skipCoverArt() {
			coverArtPath := s.saveCoverArt(cleanName, artistID, coverArt)
			_, err = tx.Exec("UPDATE albums SET cover_art_path = $1, updated_at = NOW() WHERE id = $2", coverArtPath, id)
			if err != nil {
				log.Printf("Warning: Could not update cover art for album %s: %v", cleanName, err)
			}
		}
	}

	return id, nil
}

// skipCoverArt returns whether to skip cover art processing (for optimization)
func (s *Scanner) skipCoverArt() bool {
	// Can be overridden by configuration if needed
	return false
}

// saveCoverArt saves cover art to disk and returns the file path
func (s *Scanner) saveCoverArt(albumName string, artistID int, coverArt []byte) string {
	// Create covers directory if it doesn't exist
	coversDir := "covers"
	if err := os.MkdirAll(coversDir, 0755); err != nil {
		log.Printf("Warning: Could not create covers directory: %v", err)
		return ""
	}

	// Generate filename based on album name and artist ID
	// Clean the album name to make it filesystem-safe
	cleanName := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, albumName)

	filename := fmt.Sprintf("%s_%d.jpg", cleanName, artistID)
	coverPath := filepath.Join(coversDir, filename)

	// Write cover art to file
	if err := os.WriteFile(coverPath, coverArt, 0644); err != nil {
		log.Printf("Warning: Could not save cover art to %s: %v", coverPath, err)
		return ""
	}

	log.Printf("Saved cover art for album '%s' to %s", albumName, coverPath)

	// Return path with forward slashes for cross-platform compatibility
	return filepath.ToSlash(coverPath)
}

// insertOrUpdateSong inserts or updates song information
func (s *Scanner) insertOrUpdateSong(tx *sql.Tx, file AudioFile, artistID, albumID int) error {
	// Clean the song title to handle potential encoding issues
	cleanTitle := strings.TrimSpace(file.Title)
	if cleanTitle == "" {
		cleanTitle = filepath.Base(file.Path)
	}

	// Limit title length to prevent database errors
	if len(cleanTitle) > 255 {
		cleanTitle = cleanTitle[:255]
	}

	// Ensure track number is reasonable
	trackNumber := file.TrackNumber
	if trackNumber < 0 {
		trackNumber = 0
	}
	if trackNumber > 999 {
		trackNumber = 999
	}

	// Ensure duration is reasonable (convert to seconds, max 24 hours)
	durationSeconds := int(file.Duration.Seconds())
	if durationSeconds < 0 {
		durationSeconds = 0
	}
	if durationSeconds > 86400 { // 24 hours max
		durationSeconds = 86400
	}

	// Ensure bitrate is reasonable
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

// clearExistingData removes sample data (optional)
func (s *Scanner) clearExistingData() error {
	// Delete in order to respect foreign key constraints
	queries := []string{
		"DELETE FROM favorites",
		"DELETE FROM ratings",
		"DELETE FROM play_history",
		"DELETE FROM playlist_songs",
		"DELETE FROM playlists",
		"DELETE FROM songs",
		"DELETE FROM albums",
		"DELETE FROM artists",
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("error executing %s: %v", query, err)
		}
	}

	return nil
}

// GetScanStats returns statistics about the current library
func (s *Scanner) GetScanStats() (map[string]int, error) {
	stats := make(map[string]int)

	// Count artists
	var artistCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM artists").Scan(&artistCount)
	if err != nil {
		return nil, err
	}
	stats["artists"] = artistCount

	// Count albums
	var albumCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM albums").Scan(&albumCount)
	if err != nil {
		return nil, err
	}
	stats["albums"] = albumCount

	// Count songs
	var songCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&songCount)
	if err != nil {
		return nil, err
	}
	stats["songs"] = songCount

	return stats, nil
}

// GetScanProgress returns the current progress of the library scan
func (s *Scanner) GetScanProgress() map[string]interface{} {
	progress := make(map[string]interface{})

	progress["is_scanning"] = s.IsScanning
	progress["total_files"] = s.TotalFiles
	progress["processed_files"] = s.ProcessedFiles

	// Calculate percentage
	percentComplete := 0.0
	if s.TotalFiles > 0 {
		percentComplete = float64(s.ProcessedFiles) / float64(s.TotalFiles) * 100
	}
	progress["percent_complete"] = percentComplete

	// Add last error if any
	if s.LastError != "" {
		progress["last_error"] = s.LastError
	}

	return progress
}
