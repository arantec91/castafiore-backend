package subsonic

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"castafiore-backend/internal/lastfm"

	"github.com/gin-gonic/gin"
)

// Ping - Used to test connectivity
func (s *Service) Ping(c *gin.Context) {
	s.sendResponse(c, nil)
}

// GetLicense - Returns license information
func (s *Service) GetLicense(c *gin.Context) {
	license := &License{
		Valid: true,
	}
	s.sendResponse(c, license)
}

// GetMusicFolders - Returns all configured music folders
func (s *Service) GetMusicFolders(c *gin.Context) {
	musicFolders := &MusicFolders{
		MusicFolder: []MusicFolder{
			{ID: 1, Name: "Music"},
		},
	}
	s.sendResponse(c, musicFolders)
}

// GetIndexes - Returns an indexed structure of all artists
func (s *Service) GetIndexes(c *gin.Context) {
	// Query artists from database
	rows, err := s.db.Query("SELECT id, name FROM artists ORDER BY name")
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	// Group artists by first letter
	artistMap := make(map[string][]Artist)

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}

		firstLetter := "A"
		if len(name) > 0 {
			firstLetter = string(name[0])
		}

		artist := Artist{
			ID:   strconv.Itoa(id),
			Name: name,
		}

		artistMap[firstLetter] = append(artistMap[firstLetter], artist)
	}

	// Convert map to index array
	var indexes []Index
	for letter, artists := range artistMap {
		index := Index{
			Name:   letter,
			Artist: artists,
		}
		indexes = append(indexes, index)
	}

	result := &Indexes{
		LastModified: 1640995200000, // Example timestamp
		Index:        indexes,
	}

	s.sendResponse(c, result)
}

// GetMusicDirectory - Returns a listing of all files in a music directory
func (s *Service) GetMusicDirectory(c *gin.Context) {
	id := c.Query("id")
	if !s.isValidID(id) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// For simplicity, we'll return a basic directory structure
	directory := &Directory{
		ID:   id,
		Name: "Music Directory",
		Child: []Child{
			{
				ID:          "1",
				IsDir:       false,
				Title:       "Example Song",
				Album:       "Example Album",
				Artist:      "Example Artist",
				Track:       1,
				Year:        2023,
				Genre:       "Rock",
				Size:        3145728, // 3MB
				ContentType: "audio/mpeg",
				Suffix:      "mp3",
				Duration:    180,
				BitRate:     320,
				Path:        "Example Artist/Example Album/01 - Example Song.mp3",
			},
		},
	}

	s.sendResponse(c, directory)
}

// GetGenres - Returns all genres
func (s *Service) GetGenres(c *gin.Context) {
	// Query genres from database
	rows, err := s.db.Query(`
		SELECT al.genre, COUNT(DISTINCT al.id) as album_count, COUNT(s.id) as song_count
		FROM albums al
		LEFT JOIN songs s ON al.id = s.album_id
		WHERE al.genre IS NOT NULL AND al.genre != '' AND al.genre != 'Unknown'
		GROUP BY al.genre
		ORDER BY al.genre
	`)
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	var genreList []Genre
	for rows.Next() {
		var genre Genre
		err := rows.Scan(&genre.Value, &genre.AlbumCount, &genre.SongCount)
		if err != nil {
			continue
		}
		genreList = append(genreList, genre)
	}

	// Add some default genres if none found
	if len(genreList) == 0 {
		genreList = []Genre{
			{SongCount: 0, AlbumCount: 0, Value: "Rock"},
			{SongCount: 0, AlbumCount: 0, Value: "Pop"},
			{SongCount: 0, AlbumCount: 0, Value: "Jazz"},
			{SongCount: 0, AlbumCount: 0, Value: "Classical"},
		}
	}

	genres := &Genres{
		Genre: genreList,
	}
	s.sendResponse(c, genres)
}

// GetArtists - Returns all artists
func (s *Service) GetArtists(c *gin.Context) {
	// Query artists from database
	rows, err := s.db.Query(`
		SELECT a.id, a.name, COUNT(al.id) as album_count
		FROM artists a
		LEFT JOIN albums al ON a.id = al.artist_id
		GROUP BY a.id, a.name
		ORDER BY a.name
	`)
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	// Group artists by first letter
	artistMap := make(map[string][]ArtistID3)

	for rows.Next() {
		var id int
		var name string
		var albumCount int
		if err := rows.Scan(&id, &name, &albumCount); err != nil {
			continue
		}

		firstLetter := "A"
		if len(name) > 0 {
			firstLetter = string(name[0])
		}

		artist := ArtistID3{
			ID:         strconv.Itoa(id),
			Name:       name,
			AlbumCount: albumCount,
		}

		artistMap[firstLetter] = append(artistMap[firstLetter], artist)
	}

	// Convert map to index array
	var indexes []IndexID3
	for letter, artists := range artistMap {
		index := IndexID3{
			Name:   letter,
			Artist: artists,
		}
		indexes = append(indexes, index)
	}

	result := &ArtistsID3{
		Index: indexes,
	}

	s.sendResponse(c, result)
}

// GetArtist - Returns details for an artist including a list of albums
func (s *Service) GetArtist(c *gin.Context) {
	id := c.Query("id")
	if !s.isValidID(id) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// Get artist from database
	var artist ArtistID3
	var albumCount int
	err := s.db.QueryRow(`
		SELECT ar.id, ar.name, COUNT(al.id) as album_count
		FROM artists ar
		LEFT JOIN albums al ON ar.id = al.artist_id
		WHERE ar.id = $1
		GROUP BY ar.id, ar.name
	`, id).Scan(&artist.ID, &artist.Name, &albumCount)

	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "Artist not found")
		} else {
			s.sendError(c, 0, "Database error")
		}
		return
	}

	artist.AlbumCount = albumCount

	// Get albums for this artist
	rows, err := s.db.Query(`
		SELECT id, name, year, genre, created_at, cover_art_path
		FROM albums 
		WHERE artist_id = $1
		ORDER BY year DESC, name ASC
	`, id)
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	var albums []AlbumID3
	for rows.Next() {
		var album AlbumID3
		var createdAt time.Time
		var year *int
		var genre *string
		var coverArtPath *string

		err := rows.Scan(&album.ID, &album.Name, &year, &genre, &createdAt, &coverArtPath)
		if err != nil {
			continue
		}

		album.Artist = artist.Name
		album.ArtistID = artist.ID
		album.Created = createdAt.Format("2006-01-02T15:04:05Z")

		if year != nil {
			album.Year = *year
		}
		if genre != nil {
			album.Genre = *genre
		}
		if coverArtPath != nil && *coverArtPath != "" {
			album.CoverArt = album.ID
		}

		// Get song count and duration for this album
		var songCount, totalDuration int
		s.db.QueryRow(`
			SELECT COUNT(id), COALESCE(SUM(duration), 0)
			FROM songs 
			WHERE album_id = $1
		`, album.ID).Scan(&songCount, &totalDuration)

		album.SongCount = songCount
		album.Duration = totalDuration

		albums = append(albums, album)
	}

	// Create artist with albums
	result := &ArtistWithAlbums{
		ID:         artist.ID,
		Name:       artist.Name,
		AlbumCount: artist.AlbumCount,
		Album:      albums,
	}

	s.sendResponse(c, result)
}

// GetAlbum - Returns details for an album including a list of songs
func (s *Service) GetAlbum(c *gin.Context) {
	id := c.Query("id")
	if !s.isValidID(id) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// Get album from database
	var album AlbumID3
	var artistName string
	var createdAt time.Time
	var year *int
	var genre *string
	var coverArtPath *string

	err := s.db.QueryRow(`
		SELECT al.id, al.name, al.artist_id, al.year, al.genre, al.created_at,
		       ar.name as artist_name, al.cover_art_path
		FROM albums al
		JOIN artists ar ON al.artist_id = ar.id
		WHERE al.id = $1
	`, id).Scan(&album.ID, &album.Name, &album.ArtistID, &year, &genre, &createdAt, &artistName, &coverArtPath)

	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "Album not found")
		} else {
			s.sendError(c, 0, "Database error")
		}
		return
	}

	album.Artist = artistName
	album.Created = createdAt.Format("2006-01-02T15:04:05Z")

	if year != nil {
		album.Year = *year
	}
	if genre != nil {
		album.Genre = *genre
	}
	if coverArtPath != nil && *coverArtPath != "" {
		album.CoverArt = album.ID
	}

	// Get songs for this album
	rows, err := s.db.Query(`
		SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format
		FROM songs s
		WHERE s.album_id = $1
		ORDER BY s.track_number, s.title
	`, id)
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	var songs []Child
	var totalDuration int

	for rows.Next() {
		var song Child
		var trackNumber *int

		err := rows.Scan(
			&song.ID, &song.Title, &trackNumber, &song.Duration,
			&song.Path, &song.Size, &song.BitRate, &song.Suffix,
		)
		if err != nil {
			continue
		}

		song.Album = album.Name
		song.Artist = album.Artist
		song.IsDir = false
		song.Parent = album.ID
		song.AlbumId = album.ID // Add explicit albumId field
		song.ContentType = s.getContentType(song.Suffix)

		// Set cover art to album ID if album has cover art
		if coverArtPath != nil && *coverArtPath != "" {
			song.CoverArt = album.ID
		}

		log.Printf("Debug GetAlbum: Song ID %s, Album ID %s, CoverArt %s",
			song.ID, song.Parent, song.CoverArt)

		if trackNumber != nil {
			song.Track = *trackNumber
		}
		if year != nil {
			song.Year = *year
		}
		if genre != nil {
			song.Genre = *genre
		}

		totalDuration += song.Duration
		songs = append(songs, song)
	}

	album.Song = songs
	album.SongCount = len(songs)
	album.Duration = totalDuration

	s.sendResponse(c, &album)
}

// GetSong - Returns details for a song
func (s *Service) GetSong(c *gin.Context) {
	id := c.Query("id")
	if !s.isValidID(id) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// Get song from database
	var song Child
	var trackNumber *int
	var year *int
	var genre *string
	var artistName, albumName string

	err := s.db.QueryRow(`
		SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format, s.album_id, s.year,
		       ar.name as artist_name, al.name as album_name, al.genre
		FROM songs s
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
		WHERE s.id = $1
	`, id).Scan(
		&song.ID, &song.Title, &trackNumber, &song.Duration,
		&song.Path, &song.Size, &song.BitRate, &song.Suffix,
		&song.Parent, &year, &artistName, &albumName, &genre,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "Song not found")
		} else {
			s.sendError(c, 0, "Database error")
		}
		return
	}

	song.Album = albumName
	song.Artist = artistName
	song.IsDir = false
	song.ContentType = s.getContentType(song.Suffix)

	if trackNumber != nil {
		song.Track = *trackNumber
	}
	if year != nil {
		song.Year = *year
	}
	if genre != nil {
		song.Genre = *genre
	}

	s.sendResponse(c, &song)
}

// Placeholder implementations for other endpoints
func (s *Service) GetAlbumList(c *gin.Context) { s.sendResponse(c, nil) }
func (s *Service) GetAlbumList2(c *gin.Context) {
	// Get parameters
	listType := c.DefaultQuery("type", "newest")
	size := 10 // default size
	if sizeStr := c.Query("size"); sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil && parsed > 0 {
			size = parsed
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build query based on type
	var query string
	var orderBy string

	switch listType {
	case "newest":
		orderBy = "al.created_at DESC"
	case "recent":
		orderBy = "al.updated_at DESC"
	case "frequent":
		orderBy = "al.id DESC" // Placeholder - would need play counts
	case "random":
		orderBy = "RANDOM()"
	case "alphabeticalByName":
		orderBy = "al.name ASC"
	case "alphabeticalByArtist":
		orderBy = "ar.name ASC, al.name ASC"
	case "starred":
		orderBy = "al.created_at DESC" // Placeholder - would need starred functionality
	default:
		orderBy = "al.created_at DESC"
	}

	query = fmt.Sprintf(`
		SELECT al.id, al.name, al.artist_id, al.year, al.genre, al.created_at,
		       ar.name as artist_name,
		       COUNT(s.id) as song_count,
		       COALESCE(SUM(s.duration), 0) as total_duration,
		       al.cover_art_path
		FROM albums al
		JOIN artists ar ON al.artist_id = ar.id
		LEFT JOIN songs s ON al.id = s.album_id
		GROUP BY al.id, al.name, al.artist_id, al.year, al.genre, al.created_at, ar.name, al.cover_art_path
		ORDER BY %s
		LIMIT $1 OFFSET $2
	`, orderBy)

	rows, err := s.db.Query(query, size, offset)
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	var albums []AlbumID3
	for rows.Next() {
		var album AlbumID3
		var artistName string
		var createdAt time.Time
		var songCount, totalDuration int
		var year *int
		var genre *string
		var coverArtPath *string

		err := rows.Scan(
			&album.ID, &album.Name, &album.ArtistID, &year, &genre, &createdAt,
			&artistName, &songCount, &totalDuration, &coverArtPath,
		)
		if err != nil {
			continue
		}

		album.Artist = artistName
		album.SongCount = songCount
		album.Duration = totalDuration
		album.Created = createdAt.Format("2006-01-02T15:04:05Z")

		if year != nil {
			album.Year = *year
		}
		if genre != nil {
			album.Genre = *genre
		}

		// Set cover art to album ID if album has cover art
		if coverArtPath != nil && *coverArtPath != "" {
			album.CoverArt = album.ID
		}

		albums = append(albums, album)
	}

	result := &AlbumList2{
		Album: albums,
	}

	s.sendResponse(c, result)
}
func (s *Service) GetRandomSongs(c *gin.Context) {
	// Get parameters
	size := 10 // default size
	if sizeStr := c.Query("size"); sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil && parsed > 0 && parsed <= 500 {
			size = parsed
		}
	}

	genre := c.Query("genre")
	fromYear := c.Query("fromYear")
	toYear := c.Query("toYear")
	musicFolderId := c.Query("musicFolderId")

	// Build query
	query := `
		SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format, s.album_id,
		       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
		FROM songs s
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
		WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	if genre != "" {
		argCount++
		query += fmt.Sprintf(" AND al.genre ILIKE $%d", argCount)
		args = append(args, "%"+genre+"%")
	}

	if fromYear != "" {
		if year, err := strconv.Atoi(fromYear); err == nil {
			argCount++
			query += fmt.Sprintf(" AND al.year >= $%d", argCount)
			args = append(args, year)
		}
	}

	if toYear != "" {
		if year, err := strconv.Atoi(toYear); err == nil {
			argCount++
			query += fmt.Sprintf(" AND al.year <= $%d", argCount)
			args = append(args, year)
		}
	}

	if musicFolderId != "" {
		// For now, ignore music folder filtering since we have only one folder
	}

	argCount++
	query += fmt.Sprintf(" ORDER BY RANDOM() LIMIT $%d", argCount)
	args = append(args, size)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	var songs []Child
	for rows.Next() {
		var song Child
		var trackNumber *int
		var year *int
		var genre *string
		var coverArtPath *string
		var artistName, albumName string

		err := rows.Scan(
			&song.ID, &song.Title, &trackNumber, &song.Duration,
			&song.Path, &song.Size, &song.BitRate, &song.Suffix, &song.Parent,
			&artistName, &albumName, &year, &genre, &coverArtPath,
		)
		if err != nil {
			continue
		}

		song.Album = albumName
		song.Artist = artistName
		song.IsDir = false
		song.AlbumId = song.Parent // Add explicit albumId field for GetRandomSongs
		song.ContentType = s.getContentType(song.Suffix)

		if trackNumber != nil {
			song.Track = *trackNumber
		}
		if year != nil {
			song.Year = *year
		}
		if genre != nil {
			song.Genre = *genre
		}

		// Set cover art to album ID if album has cover art
		if coverArtPath != nil && *coverArtPath != "" {
			song.CoverArt = song.Parent
		}

		log.Printf("Debug GetRandomSongs: Song ID %s, Album ID %s, CoverArt %s",
			song.ID, song.Parent, song.CoverArt)

		songs = append(songs, song)
	}

	result := &RandomSongs{
		Song: songs,
	}

	s.sendResponse(c, result)
}
func (s *Service) GetSongsByGenre(c *gin.Context) {
	genre := c.Query("genre")
	if genre == "" {
		s.sendError(c, 10, "Required parameter is missing")
		return
	}

	count := 10 // default count
	if countStr := c.Query("count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 && parsed <= 500 {
			count = parsed
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	musicFolderId := c.Query("musicFolderId")

	query := `
		SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format, s.album_id,
		       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
		FROM songs s
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
		WHERE al.genre ILIKE $1
		ORDER BY ar.name, al.name, s.track_number
		LIMIT $2 OFFSET $3`

	args := []interface{}{"%" + genre + "%", count, offset}

	if musicFolderId != "" {
		// For now, ignore music folder filtering since we have only one folder
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	var songs []Child
	for rows.Next() {
		var song Child
		var trackNumber *int
		var year *int
		var genre *string
		var coverArtPath *string
		var artistName, albumName string

		err := rows.Scan(
			&song.ID, &song.Title, &trackNumber, &song.Duration,
			&song.Path, &song.Size, &song.BitRate, &song.Suffix, &song.Parent,
			&artistName, &albumName, &year, &genre, &coverArtPath,
		)
		if err != nil {
			continue
		}

		song.Album = albumName
		song.Artist = artistName
		song.IsDir = false
		song.AlbumId = song.Parent // Add explicit albumId field for GetSongsByGenre
		song.ContentType = s.getContentType(song.Suffix)

		if trackNumber != nil {
			song.Track = *trackNumber
		}
		if year != nil {
			song.Year = *year
		}
		if genre != nil {
			song.Genre = *genre
		}

		// Set cover art to album ID if album has cover art
		if coverArtPath != nil && *coverArtPath != "" {
			song.CoverArt = song.Parent
		}

		log.Printf("Debug GetSongsByGenre: Song ID %s, Album ID %s, CoverArt %s",
			song.ID, song.Parent, song.CoverArt)

		songs = append(songs, song)
	}

	result := &SongsByGenre{
		Song: songs,
	}

	s.sendResponse(c, result)
}
func (s *Service) GetTopSongs(c *gin.Context) {
	// Get artist parameter
	artist := c.Query("artist")
	count := 50 // default count
	if countStr := c.Query("count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 {
			count = parsed
		}
	}

	var songs []Child
	var query string
	var args []interface{}

	if artist != "" {
		// Get top songs for specific artist
		query = `
			SELECT s.id, s.title, s.artist_id, s.album_id, s.track_number, s.duration, 
			       s.file_path, s.file_size, s.bitrate, s.format,
			       ar.name as artist_name, al.name as album_name, al.year, al.cover_art_path
			FROM songs s
			JOIN artists ar ON s.artist_id = ar.id
			JOIN albums al ON s.album_id = al.id
			WHERE ar.name ILIKE $1
			ORDER BY s.track_number, s.title
			LIMIT $2`
		args = []interface{}{"%" + artist + "%", count}
	} else {
		// Get random top songs from all artists
		query = `
			SELECT s.id, s.title, s.artist_id, s.album_id, s.track_number, s.duration, 
			       s.file_path, s.file_size, s.bitrate, s.format,
			       ar.name as artist_name, al.name as album_name, al.year, al.cover_art_path
			FROM songs s
			JOIN artists ar ON s.artist_id = ar.id
			JOIN albums al ON s.album_id = al.id
			ORDER BY RANDOM()
			LIMIT $1`
		args = []interface{}{count}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		s.sendError(c, 0, "Database error")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var song Child
		var artistName, albumName string
		var year *int
		var coverArtPath *string
		var artistID, albumID int

		err := rows.Scan(
			&song.ID, &song.Title, &artistID, &albumID, &song.Track,
			&song.Duration, &song.Path, &song.Size, &song.BitRate, &song.Suffix,
			&artistName, &albumName, &year, &coverArtPath,
		)
		if err != nil {
			continue
		}

		song.Artist = artistName
		song.Album = albumName
		if year != nil {
			song.Year = *year
		}
		song.ContentType = s.getContentType(song.Suffix)
		song.IsDir = false
		song.Parent = strconv.Itoa(albumID)
		song.AlbumId = strconv.Itoa(albumID) // Add explicit albumId field for GetTopSongs

		// Set cover art to album ID if album has cover art
		if coverArtPath != nil && *coverArtPath != "" {
			song.CoverArt = song.Parent
		}

		log.Printf("Debug GetTopSongs: Song ID %s, Album ID %s, CoverArt %s",
			song.ID, song.Parent, song.CoverArt)

		songs = append(songs, song)
	}

	result := &TopSongs{
		Song: songs,
	}

	s.sendResponse(c, result)
}

// GetNowPlaying - Returns what is currently being played by all users
func (s *Service) GetNowPlaying(c *gin.Context) {
	// Get all currently playing songs (updated in last 5 minutes)
	rows, err := s.db.Query(`
		SELECT np.user_id, np.song_id, np.player_id, np.started_at,
		       u.username,
		       s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format, s.album_id,
		       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
		FROM now_playing np
		JOIN users u ON np.user_id = u.id
		JOIN songs s ON np.song_id = s.id
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
		WHERE np.updated_at > NOW() - INTERVAL '5 minutes'
		ORDER BY np.updated_at DESC
	`)

	if err != nil {
		log.Printf("Error getting now playing: %v", err)
		s.sendResponse(c, &NowPlaying{Entry: []NowPlayingEntry{}})
		return
	}
	defer rows.Close()

	var entries []NowPlayingEntry
	for rows.Next() {
		var entry NowPlayingEntry
		var userId int
		var songId int
		var startedAt time.Time
		var trackNumber *int
		var year *int
		var genre *string
		var coverArtPath *string

		err := rows.Scan(
			&userId, &songId, &entry.PlayerId, &startedAt,
			&entry.Username,
			&entry.ID, &entry.Title, &trackNumber, &entry.Duration, &entry.Path,
			&entry.Size, &entry.BitRate, &entry.Suffix, &entry.AlbumId,
			&entry.Artist, &entry.Album, &year, &genre, &coverArtPath,
		)

		if err != nil {
			continue
		}

		entry.IsDir = false
		entry.ContentType = s.getContentType(entry.Suffix)

		if trackNumber != nil {
			entry.Track = *trackNumber
		}
		if year != nil {
			entry.Year = *year
		}
		if genre != nil {
			entry.Genre = *genre
		}
		if coverArtPath != nil && *coverArtPath != "" {
			entry.CoverArt = entry.AlbumId
		}

		// Calculate minutes ago
		minutesAgo := int(time.Since(startedAt).Minutes())
		entry.MinutesAgo = minutesAgo

		entries = append(entries, entry)
	}

	result := &NowPlaying{
		Entry: entries,
	}

	s.sendResponse(c, result)
}

// GetStarred - Returns starred songs, albums and artists (old format)
func (s *Service) GetStarred(c *gin.Context) {
	// Get user ID from context (with fallback to user 1 for now)
	userId := s.getUserID(c)

	result := &Starred{
		Artist: []Artist{},
		Album:  []Child{},
		Song:   []Child{},
	}

	// Get starred artists
	log.Printf("GetStarred: Fetching starred artists for user %d", userId)
	artistRows, err := s.db.Query(`
		SELECT a.id, a.name
		FROM starred_artists sa
		JOIN artists a ON sa.artist_id = a.id
		WHERE sa.user_id = $1
		ORDER BY sa.starred_at DESC
	`, userId)

	if err != nil {
		log.Printf("Error fetching starred artists: %v", err)
	} else {
		defer artistRows.Close()
		for artistRows.Next() {
			var artist Artist
			if err := artistRows.Scan(&artist.ID, &artist.Name); err == nil {
				result.Artist = append(result.Artist, artist)
			} else {
				log.Printf("Error scanning starred artist: %v", err)
			}
		}
	}

	// Get starred albums (as Child objects)
	log.Printf("GetStarred: Fetching starred albums for user %d", userId)
	albumRows, err := s.db.Query(`
		SELECT al.id, al.name, ar.id as artist_id, ar.name as artist_name, 
		       al.year, al.genre, al.cover_art_path
		FROM starred_albums sa
		JOIN albums al ON sa.album_id = al.id
		JOIN artists ar ON al.artist_id = ar.id
		WHERE sa.user_id = $1
		ORDER BY sa.starred_at DESC
	`, userId)

	if err != nil {
		log.Printf("Error fetching starred albums: %v", err)
	} else {
		defer albumRows.Close()
		for albumRows.Next() {
			var album Child
			var albumName string
			var artistId int
			var year *int
			var genre *string
			var coverArtPath *string

			if err := albumRows.Scan(&album.ID, &albumName, &artistId, &album.Artist, &year, &genre, &coverArtPath); err == nil {
				album.IsDir = true
				album.Title = albumName
				album.Album = albumName
				album.AlbumId = album.ID              // Set albumId to the album's own ID
				album.Parent = strconv.Itoa(artistId) // Set parent to artist ID
				if year != nil {
					album.Year = *year
				}
				if genre != nil {
					album.Genre = *genre
				}
				if coverArtPath != nil && *coverArtPath != "" {
					album.CoverArt = album.ID
				}
				log.Printf("DEBUG: Adding starred album - ID: %s, AlbumId: %s, Title: %s, Album: %s, Artist: %s, Parent: %s, IsDir: %t",
					album.ID, album.AlbumId, album.Title, album.Album, album.Artist, album.Parent, album.IsDir)
				result.Album = append(result.Album, album)
			} else {
				log.Printf("Error scanning starred album: %v", err)
			}
		}
	}

	// Get starred songs
	log.Printf("GetStarred: Fetching starred songs for user %d", userId)
	songRows, err := s.db.Query(`
		SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format, s.album_id,
		       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
		FROM starred_songs ss
		JOIN songs s ON ss.song_id = s.id
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
		WHERE ss.user_id = $1
		ORDER BY ss.starred_at DESC
	`, userId)

	if err != nil {
		log.Printf("Error fetching starred songs: %v", err)
	} else {
		defer songRows.Close()
		result.Song = s.scanSongs(songRows)
	}

	log.Printf("GetStarred: Returning %d artists, %d albums, %d songs for user %d",
		len(result.Artist), len(result.Album), len(result.Song), userId)
	s.sendResponse(c, result)
}

// GetStarred2 - Returns starred songs, albums and artists (ID3 format)
func (s *Service) GetStarred2(c *gin.Context) {
	// Get user ID from context (with fallback to user 1 for now)
	userId := s.getUserID(c)

	result := &Starred2{
		Artist: []ArtistID3{},
		Album:  []AlbumID3{},
		Song:   []Child{},
	}

	// Get starred artists (ID3 format)
	log.Printf("GetStarred2: Fetching starred artists for user %d", userId)
	artistRows, err := s.db.Query(`
		SELECT a.id, a.name, COUNT(al.id) as album_count, a.cover_art_path
		FROM starred_artists sa
		JOIN artists a ON sa.artist_id = a.id
		LEFT JOIN albums al ON a.id = al.artist_id
		WHERE sa.user_id = $1
		GROUP BY a.id, a.name, a.cover_art_path
		ORDER BY sa.starred_at DESC
	`, userId)

	if err != nil {
		log.Printf("Error fetching starred artists for ID3: %v", err)
	} else {
		defer artistRows.Close()
		for artistRows.Next() {
			var artist ArtistID3
			var coverArtPath *string
			if err := artistRows.Scan(&artist.ID, &artist.Name, &artist.AlbumCount, &coverArtPath); err == nil {
				if coverArtPath != nil && *coverArtPath != "" {
					artist.CoverArt = artist.ID
				}
				result.Artist = append(result.Artist, artist)
			} else {
				log.Printf("Error scanning starred artist ID3: %v", err)
			}
		}
	}

	// Get starred albums (ID3 format)
	log.Printf("GetStarred2: Fetching starred albums for user %d", userId)
	albumRows, err := s.db.Query(`
		SELECT al.id, al.name, al.artist_id, ar.name as artist_name, 
		       al.year, al.genre, al.created_at, al.cover_art_path,
		       COUNT(s.id) as song_count, COALESCE(SUM(s.duration), 0) as total_duration
		FROM starred_albums sa
		JOIN albums al ON sa.album_id = al.id
		JOIN artists ar ON al.artist_id = ar.id
		LEFT JOIN songs s ON al.id = s.album_id
		WHERE sa.user_id = $1
		GROUP BY al.id, al.name, al.artist_id, ar.name, al.year, al.genre, al.created_at, al.cover_art_path
		ORDER BY sa.starred_at DESC
	`, userId)

	if err != nil {
		log.Printf("Error fetching starred albums for ID3: %v", err)
	} else {
		defer albumRows.Close()
		for albumRows.Next() {
			var album AlbumID3
			var createdAt time.Time
			var year *int
			var genre *string
			var coverArtPath *string

			if err := albumRows.Scan(&album.ID, &album.Name, &album.ArtistID, &album.Artist,
				&year, &genre, &createdAt, &coverArtPath, &album.SongCount, &album.Duration); err == nil {
				album.Created = createdAt.Format("2006-01-02T15:04:05Z")
				if year != nil {
					album.Year = *year
				}
				if genre != nil {
					album.Genre = *genre
				}
				if coverArtPath != nil && *coverArtPath != "" {
					album.CoverArt = album.ID
				}
				result.Album = append(result.Album, album)
			} else {
				log.Printf("Error scanning starred album ID3: %v", err)
			}
		}
	}

	// Get starred songs
	log.Printf("GetStarred2: Fetching starred songs for user %d", userId)
	songRows, err := s.db.Query(`
		SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format, s.album_id,
		       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
		FROM starred_songs ss
		JOIN songs s ON ss.song_id = s.id
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
		WHERE ss.user_id = $1
		ORDER BY ss.starred_at DESC
	`, userId)

	if err != nil {
		log.Printf("Error fetching starred songs for ID3: %v", err)
	} else {
		defer songRows.Close()
		result.Song = s.scanSongs(songRows)
	}

	log.Printf("GetStarred2: Returning %d artists, %d albums, %d songs for user %d",
		len(result.Artist), len(result.Album), len(result.Song), userId)
	s.sendResponse(c, result)
}

func (s *Service) Search2(c *gin.Context) { s.sendResponse(c, nil) }
func (s *Service) Search3(c *gin.Context) {
	// Obtener parámetros de búsqueda
	query := c.Query("query")

	// Obtener límites opcionales
	artistCount := parseIntDefault(c.Query("artistCount"), 20)
	albumCount := parseIntDefault(c.Query("albumCount"), 20)
	songCount := parseIntDefault(c.Query("songCount"), 20)

	// Obtener offsets opcionales
	artistOffset := parseIntDefault(c.Query("artistOffset"), 0)
	albumOffset := parseIntDefault(c.Query("albumOffset"), 0)
	songOffset := parseIntDefault(c.Query("songOffset"), 0)

	// For future use when implementing multiple music folders
	_ = c.Query("musicFolderId")

	result := &SearchResult3{
		Artist: []ArtistID3{},
		Album:  []AlbumID3{},
		Song:   []Child{},
	}

	// Si no hay query, usar búsqueda amplia para mostrar todo el contenido
	var searchTerm string
	if query == "" {
		searchTerm = "%"
	} else {
		searchTerm = "%" + strings.ToLower(query) + "%"
	}

	// Search artists - solo buscar si se solicitan artistas
	if artistCount > 0 {
		artistQuery := `
			SELECT ar.id, ar.name, COUNT(al.id) as album_count
			FROM artists ar
			LEFT JOIN albums al ON ar.id = al.artist_id
			WHERE LOWER(ar.name) LIKE $1
			GROUP BY ar.id, ar.name
			ORDER BY ar.name
			LIMIT $2 OFFSET $3`

		rows, err := s.db.Query(artistQuery, searchTerm, artistCount, artistOffset)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var artist ArtistID3
				var albumCount int
				err := rows.Scan(&artist.ID, &artist.Name, &albumCount)
				if err == nil {
					artist.AlbumCount = albumCount
					result.Artist = append(result.Artist, artist)
				}
			}
			rows.Close()
		}
	}

	// Search albums - solo buscar si se solicitan álbumes
	if albumCount > 0 {
		albumQuery := `
			SELECT al.id, al.name, al.artist_id, al.year, al.genre, al.created_at,
			       ar.name as artist_name,
			       COUNT(s.id) as song_count,
			       COALESCE(SUM(s.duration), 0) as total_duration,
			       al.cover_art_path
			FROM albums al
			JOIN artists ar ON al.artist_id = ar.id
			LEFT JOIN songs s ON al.id = s.album_id
			WHERE LOWER(al.name) LIKE $1 OR LOWER(ar.name) LIKE $1
			GROUP BY al.id, al.name, al.artist_id, al.year, al.genre, al.created_at, ar.name, al.cover_art_path
			ORDER BY ar.name, al.name
			LIMIT $2 OFFSET $3`

		rows, err := s.db.Query(albumQuery, searchTerm, albumCount, albumOffset)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var album AlbumID3
				var artistName string
				var createdAt time.Time
				var songCount, totalDuration int
				var year *int
				var genre *string
				var coverArtPath *string

				err := rows.Scan(
					&album.ID, &album.Name, &album.ArtistID, &year, &genre, &createdAt,
					&artistName, &songCount, &totalDuration, &coverArtPath,
				)
				if err == nil {
					album.Artist = artistName
					album.SongCount = songCount
					album.Duration = totalDuration
					album.Created = createdAt.Format("2006-01-02T15:04:05Z")

					if year != nil {
						album.Year = *year
					}
					if genre != nil {
						album.Genre = *genre
					}

					// Set cover art to album ID if album has cover art
					if coverArtPath != nil && *coverArtPath != "" {
						album.CoverArt = album.ID
					}

					result.Album = append(result.Album, album)
				}
			}
			rows.Close()
		}
	} // Search songs - solo buscar si se solicitan canciones
	if songCount > 0 {
		songQuery := `
				SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
				       s.file_size, s.bitrate, s.format, s.album_id,
				       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
				FROM songs s
				JOIN artists ar ON s.artist_id = ar.id
				JOIN albums al ON s.album_id = al.id
				WHERE LOWER(s.title) LIKE $1 OR LOWER(ar.name) LIKE $1 OR LOWER(al.name) LIKE $1
				ORDER BY ar.name, al.name, s.track_number
				LIMIT $2 OFFSET $3`

		rows, err := s.db.Query(songQuery, searchTerm, songCount, songOffset)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var song Child
				var trackNumber *int
				var year *int
				var genre *string
				var coverArtPath *string
				var artistName, albumName string
				var albumID int

				err := rows.Scan(
					&song.ID, &song.Title, &trackNumber, &song.Duration,
					&song.Path, &song.Size, &song.BitRate, &song.Suffix, &albumID,
					&artistName, &albumName, &year, &genre, &coverArtPath,
				)
				if err == nil {
					// Convert albumID to string for Parent field
					song.Parent = strconv.Itoa(albumID)
					song.AlbumId = strconv.Itoa(albumID) // Add explicit albumId field
					song.Album = albumName
					song.Artist = artistName
					song.IsDir = false
					song.ContentType = s.getContentType(song.Suffix)

					if trackNumber != nil {
						song.Track = *trackNumber
					}
					if year != nil {
						song.Year = *year
					}
					if genre != nil {
						song.Genre = *genre
					}

					// Set cover art to album ID if album has cover art
					if coverArtPath != nil && *coverArtPath != "" {
						song.CoverArt = song.Parent // This will be the album ID as string
					}

					// Asegurar que duration tenga un valor por defecto si es 0
					if song.Duration == 0 {
						song.Duration = 180 // 3 minutos por defecto
					}

					// Asegurar que bitRate tenga un valor por defecto si es 0
					if song.BitRate == 0 {
						song.BitRate = 128 // 128kbps por defecto
					}

					// Debug: verificar que el campo duration se está asignando
					log.Printf("Debug Search3: Song ID %s, Album ID %s, CoverArt %s, Duration: %d, BitRate: %d",
						song.ID, song.Parent, song.CoverArt, song.Duration, song.BitRate)

					result.Song = append(result.Song, song)
				}
			}
			rows.Close()
		}
	}

	s.sendResponse(c, result)
}

// GetPlaylists - Returns all playlists a user is allowed to play
func (s *Service) GetPlaylists(c *gin.Context) {
	// Get user from context (would be set by auth middleware)
	// For now, use a default user ID
	userId := 1 // TODO: Get from authenticated user context
	username := c.Query("username")
	if username == "" {
		username = "admin" // TODO: Get from authenticated user
	}

	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.comment, p.is_public, p.created_at, p.updated_at,
		       u.username as owner,
		       COUNT(ps.id) as song_count,
		       COALESCE(SUM(s.duration), 0) as total_duration
		FROM playlists p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN playlist_songs ps ON p.id = ps.playlist_id
		LEFT JOIN songs s ON ps.song_id = s.id
		WHERE p.user_id = $1 OR p.is_public = true
		GROUP BY p.id, p.name, p.comment, p.is_public, p.created_at, p.updated_at, u.username
		ORDER BY p.name
	`, userId)

	if err != nil {
		log.Printf("Error getting playlists: %v", err)
		s.sendResponse(c, &Playlists{Playlist: []PlaylistID3{}})
		return
	}
	defer rows.Close()

	var playlists []PlaylistID3
	for rows.Next() {
		var playlist PlaylistID3
		var comment *string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&playlist.ID, &playlist.Name, &comment, &playlist.Public,
			&createdAt, &updatedAt, &playlist.Owner,
			&playlist.SongCount, &playlist.Duration,
		)

		if err != nil {
			continue
		}

		if comment != nil {
			playlist.Comment = *comment
		}
		playlist.Created = createdAt.Format("2006-01-02T15:04:05Z")
		playlist.Changed = updatedAt.Format("2006-01-02T15:04:05Z")

		playlists = append(playlists, playlist)
	}

	result := &Playlists{
		Playlist: playlists,
	}

	s.sendResponse(c, result)
}

// GetPlaylist - Returns a listing of files in a saved playlist
func (s *Service) GetPlaylist(c *gin.Context) {
	id := c.Query("id")
	if !s.isValidID(id) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// Get playlist info
	var playlist PlaylistWithSongs
	var comment *string
	var createdAt, updatedAt time.Time
	var userId int

	err := s.db.QueryRow(`
		SELECT p.id, p.name, p.comment, p.is_public, p.created_at, p.updated_at,
		       u.username as owner, p.user_id
		FROM playlists p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = $1
	`, id).Scan(&playlist.ID, &playlist.Name, &comment, &playlist.Public,
		&createdAt, &updatedAt, &playlist.Owner, &userId)

	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "Playlist not found")
		} else {
			s.sendError(c, 0, "Database error")
		}
		return
	}

	if comment != nil {
		playlist.Comment = *comment
	}
	playlist.Created = createdAt.Format("2006-01-02T15:04:05Z")
	playlist.Changed = updatedAt.Format("2006-01-02T15:04:05Z")

	// Get songs in playlist
	rows, err := s.db.Query(`
		SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format, s.album_id,
		       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
		FROM playlist_songs ps
		JOIN songs s ON ps.song_id = s.id
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
		WHERE ps.playlist_id = $1
		ORDER BY ps.position, ps.added_at
	`, id)

	if err != nil {
		log.Printf("Error getting playlist songs: %v", err)
		playlist.Entry = []Child{}
	} else {
		defer rows.Close()
		playlist.Entry = s.scanSongs(rows)
	}

	playlist.SongCount = len(playlist.Entry)
	totalDuration := 0
	for _, song := range playlist.Entry {
		totalDuration += song.Duration
	}
	playlist.Duration = totalDuration

	s.sendResponse(c, &playlist)
}

// CreatePlaylist - Creates a new playlist
func (s *Service) CreatePlaylist(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		s.sendError(c, 10, "Required parameter 'name' is missing")
		return
	}

	// Get user from context
	userId := 1 // TODO: Get from authenticated user context

	comment := c.Query("comment")

	// Get song IDs to add
	songIds := c.QueryArray("songId")

	// Create playlist
	var playlistId int
	err := s.db.QueryRow(`
		INSERT INTO playlists (user_id, name, comment, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, false, NOW(), NOW())
		RETURNING id
	`, userId, name, comment).Scan(&playlistId)

	if err != nil {
		log.Printf("Error creating playlist: %v", err)
		s.sendError(c, 0, "Failed to create playlist")
		return
	}

	// Add songs to playlist
	for i, songId := range songIds {
		_, err := s.db.Exec(`
			INSERT INTO playlist_songs (playlist_id, song_id, position, added_at)
			VALUES ($1, $2, $3, NOW())
		`, playlistId, songId, i)

		if err != nil {
			log.Printf("Error adding song %s to playlist: %v", songId, err)
		}
	}

	s.sendResponse(c, nil)
}

// UpdatePlaylist - Updates a playlist
func (s *Service) UpdatePlaylist(c *gin.Context) {
	playlistId := c.Query("playlistId")
	if !s.isValidID(playlistId) {
		s.sendError(c, 10, "Required parameter 'playlistId' is missing or invalid")
		return
	}

	// Get user from context
	userId := 1 // TODO: Get from authenticated user context

	// Check if user owns the playlist
	var ownerId int
	err := s.db.QueryRow("SELECT user_id FROM playlists WHERE id = $1", playlistId).Scan(&ownerId)
	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "Playlist not found")
		} else {
			s.sendError(c, 0, "Database error")
		}
		return
	}

	if ownerId != userId {
		s.sendError(c, 50, "User is not authorized to update this playlist")
		return
	}

	// Update playlist metadata
	name := c.Query("name")
	comment := c.Query("comment")
	isPublic := c.Query("public")

	if name != "" {
		_, err := s.db.Exec("UPDATE playlists SET name = $1, updated_at = NOW() WHERE id = $2", name, playlistId)
		if err != nil {
			log.Printf("Error updating playlist name: %v", err)
		}
	}

	if comment != "" {
		_, err := s.db.Exec("UPDATE playlists SET comment = $1, updated_at = NOW() WHERE id = $2", comment, playlistId)
		if err != nil {
			log.Printf("Error updating playlist comment: %v", err)
		}
	}

	if isPublic != "" {
		publicBool := isPublic == "true"
		_, err := s.db.Exec("UPDATE playlists SET is_public = $1, updated_at = NOW() WHERE id = $2", publicBool, playlistId)
		if err != nil {
			log.Printf("Error updating playlist visibility: %v", err)
		}
	}

	// Add song to playlist
	songIdToAdd := c.Query("songIdToAdd")
	if songIdToAdd != "" {
		// Get max position
		var maxPosition int
		s.db.QueryRow("SELECT COALESCE(MAX(position), -1) FROM playlist_songs WHERE playlist_id = $1", playlistId).Scan(&maxPosition)

		_, err := s.db.Exec(`
			INSERT INTO playlist_songs (playlist_id, song_id, position, added_at)
			VALUES ($1, $2, $3, NOW())
		`, playlistId, songIdToAdd, maxPosition+1)

		if err != nil {
			log.Printf("Error adding song to playlist: %v", err)
		}
	}

	// Remove song from playlist
	songIndexToRemove := c.Query("songIndexToRemove")
	if songIndexToRemove != "" {
		index, err := strconv.Atoi(songIndexToRemove)
		if err == nil {
			// Delete song at position
			_, err := s.db.Exec(`
				DELETE FROM playlist_songs 
				WHERE playlist_id = $1 AND position = $2
			`, playlistId, index)

			if err != nil {
				log.Printf("Error removing song from playlist: %v", err)
			}
		}
	}

	s.sendResponse(c, nil)
}

// DeletePlaylist - Deletes a saved playlist
func (s *Service) DeletePlaylist(c *gin.Context) {
	id := c.Query("id")
	if !s.isValidID(id) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// Get user from context
	userId := 1 // TODO: Get from authenticated user context

	// Check if user owns the playlist
	var ownerId int
	err := s.db.QueryRow("SELECT user_id FROM playlists WHERE id = $1", id).Scan(&ownerId)
	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "Playlist not found")
		} else {
			s.sendError(c, 0, "Database error")
		}
		return
	}

	if ownerId != userId {
		s.sendError(c, 50, "User is not authorized to delete this playlist")
		return
	}

	// Delete playlist (cascade will delete playlist_songs)
	_, err = s.db.Exec("DELETE FROM playlists WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting playlist: %v", err)
		s.sendError(c, 0, "Failed to delete playlist")
		return
	}

	s.sendResponse(c, nil)
}
func (s *Service) GetCoverArt(c *gin.Context) {
	id := c.Query("id")
	size := c.Query("size") // Optional size parameter

	log.Printf("GetCoverArt request: id=%s, size=%s", id, size)

	if !s.isValidID(id) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// The ID can be either an album ID or a song ID
	// First, try to get cover art from album
	var coverArtPath sql.NullString
	err := s.db.QueryRow("SELECT cover_art_path FROM albums WHERE id = $1", id).Scan(&coverArtPath)

	if err == sql.ErrNoRows {
		// Try to get cover art from song's album
		err = s.db.QueryRow(`
			SELECT a.cover_art_path 
			FROM songs s 
			JOIN albums a ON s.album_id = a.id 
			WHERE s.id = $1
		`, id).Scan(&coverArtPath)
	}

	if err != nil || !coverArtPath.Valid || coverArtPath.String == "" {
		// No cover art found, return 404
		log.Printf("No cover art found for ID %s: err=%v, valid=%v, path=%s", id, err, coverArtPath.Valid, coverArtPath.String)
		c.Status(http.StatusNotFound)
		return
	}

	// Ensure the cover art path is absolute or relative to working directory
	coverPath := coverArtPath.String
	if !filepath.IsAbs(coverPath) {
		// Normalize path separators for the current OS
		coverPath = filepath.FromSlash(coverPath)
		// Make relative paths relative to current working directory
		coverPath = filepath.Join(".", coverPath)
	}

	// Read the cover art file
	coverData, err := os.ReadFile(coverPath)
	if err != nil {
		log.Printf("Error reading cover art file %s: %v", coverPath, err)
		// Try to check if file exists with more detailed logging
		if _, statErr := os.Stat(coverPath); os.IsNotExist(statErr) {
			log.Printf("Cover art file does not exist: %s", coverPath)
		} else {
			log.Printf("Cover art file exists but cannot be read: %s, stat error: %v", coverPath, statErr)
		}
		c.Status(http.StatusNotFound)
		return
	}

	// Determine content type based on file extension
	contentType := "image/jpeg"
	ext := strings.ToLower(filepath.Ext(coverArtPath.String))
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	// Set headers and send the image
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.Itoa(len(coverData)))
	c.Header("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.Header("Accept-Ranges", "bytes")

	log.Printf("Serving cover art: id=%s, path=%s, size=%d bytes, content-type=%s", id, coverPath, len(coverData), contentType)

	// TODO: Handle size parameter for image resizing if needed
	// For now, we serve the original image regardless of size parameter

	c.Data(http.StatusOK, contentType, coverData)
}
func (s *Service) GetLyrics(c *gin.Context) { s.sendResponse(c, nil) }
func (s *Service) GetAvatar(c *gin.Context) { c.Status(http.StatusNotFound) }

// GetUser - Returns details about a given user
func (s *Service) GetUser(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		s.sendError(c, 10, "Required parameter 'username' is missing")
		return
	}

	var user User
	var email string
	var isAdmin bool

	err := s.db.QueryRow(`
		SELECT username, email, is_admin
		FROM users
		WHERE username = $1
	`, username).Scan(&user.Username, &email, &isAdmin)

	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "User not found")
		} else {
			s.sendError(c, 0, "Database error")
		}
		return
	}

	user.Email = email
	user.AdminRole = isAdmin
	user.ScrobblingEnabled = true
	user.SettingsRole = isAdmin
	user.DownloadRole = true
	user.UploadRole = isAdmin
	user.PlaylistRole = true
	user.CoverArtRole = true
	user.CommentRole = true
	user.PodcastRole = true
	user.StreamRole = true
	user.JukeboxRole = false
	user.ShareRole = true
	user.VideoConversionRole = false

	s.sendResponse(c, &user)
}

func (s *Service) GetUsers(c *gin.Context)       { s.sendResponse(c, nil) }
func (s *Service) CreateUser(c *gin.Context)     { s.sendResponse(c, nil) }
func (s *Service) UpdateUser(c *gin.Context)     { s.sendResponse(c, nil) }
func (s *Service) DeleteUser(c *gin.Context)     { s.sendResponse(c, nil) }
func (s *Service) ChangePassword(c *gin.Context) { s.sendResponse(c, nil) }

// Star - Attaches a star to a song, album or artist
func (s *Service) Star(c *gin.Context) {
	// Get user ID from context (with fallback to user 1 for now)
	userId := s.getUserID(c)

	// Get IDs to star
	songIds := c.QueryArray("id")
	albumIds := c.QueryArray("albumId")
	artistIds := c.QueryArray("artistId")

	var errors []string

	// Star songs
	for _, songId := range songIds {
		if !s.isValidID(songId) {
			log.Printf("Invalid song ID provided: %s", songId)
			continue
		}

		// Check if song exists
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM songs WHERE id = $1)", songId).Scan(&exists)
		if err != nil || !exists {
			log.Printf("Song with ID %s does not exist", songId)
			errors = append(errors, fmt.Sprintf("Song %s not found", songId))
			continue
		}

		_, err = s.db.Exec(`
			INSERT INTO starred_songs (user_id, song_id, starred_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (user_id, song_id) DO NOTHING
		`, userId, songId)

		if err != nil {
			log.Printf("Error starring song %s: %v", songId, err)
			errors = append(errors, fmt.Sprintf("Failed to star song %s", songId))
		} else {
			log.Printf("Successfully starred song %s for user %d", songId, userId)
		}
	}

	// Star albums
	for _, albumId := range albumIds {
		if !s.isValidID(albumId) {
			log.Printf("Invalid album ID provided: %s", albumId)
			continue
		}

		// Check if album exists
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM albums WHERE id = $1)", albumId).Scan(&exists)
		if err != nil || !exists {
			log.Printf("Album with ID %s does not exist", albumId)
			errors = append(errors, fmt.Sprintf("Album %s not found", albumId))
			continue
		}

		_, err = s.db.Exec(`
			INSERT INTO starred_albums (user_id, album_id, starred_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (user_id, album_id) DO NOTHING
		`, userId, albumId)

		if err != nil {
			log.Printf("Error starring album %s: %v", albumId, err)
			errors = append(errors, fmt.Sprintf("Failed to star album %s", albumId))
		} else {
			log.Printf("Successfully starred album %s for user %d", albumId, userId)
		}
	}

	// Star artists
	for _, artistId := range artistIds {
		if !s.isValidID(artistId) {
			log.Printf("Invalid artist ID provided: %s", artistId)
			continue
		}

		// Check if artist exists
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM artists WHERE id = $1)", artistId).Scan(&exists)
		if err != nil || !exists {
			log.Printf("Artist with ID %s does not exist", artistId)
			errors = append(errors, fmt.Sprintf("Artist %s not found", artistId))
			continue
		}

		_, err = s.db.Exec(`
			INSERT INTO starred_artists (user_id, artist_id, starred_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (user_id, artist_id) DO NOTHING
		`, userId, artistId)

		if err != nil {
			log.Printf("Error starring artist %s: %v", artistId, err)
			errors = append(errors, fmt.Sprintf("Failed to star artist %s", artistId))
		} else {
			log.Printf("Successfully starred artist %s for user %d", artistId, userId)
		}
	}

	// If there were critical errors, send an error response
	if len(errors) > 0 {
		log.Printf("Star operation completed with %d errors", len(errors))
	}

	// Send successful response (Subsonic API doesn't send error details for star operations)
	s.sendResponse(c, nil)
}

// Unstar - Removes the star from a song, album or artist
func (s *Service) Unstar(c *gin.Context) {
	// Get user ID from context (with fallback to user 1 for now)
	userId := s.getUserID(c)

	// Get IDs to unstar
	songIds := c.QueryArray("id")
	albumIds := c.QueryArray("albumId")
	artistIds := c.QueryArray("artistId")

	var errors []string

	// Unstar songs
	for _, songId := range songIds {
		if !s.isValidID(songId) {
			log.Printf("Invalid song ID provided: %s", songId)
			continue
		}

		result, err := s.db.Exec(`
			DELETE FROM starred_songs
			WHERE user_id = $1 AND song_id = $2
		`, userId, songId)

		if err != nil {
			log.Printf("Error unstarring song %s: %v", songId, err)
			errors = append(errors, fmt.Sprintf("Failed to unstar song %s", songId))
		} else {
			if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
				log.Printf("Successfully unstarred song %s for user %d", songId, userId)
			} else {
				log.Printf("Song %s was not starred by user %d", songId, userId)
			}
		}
	}

	// Unstar albums
	for _, albumId := range albumIds {
		if !s.isValidID(albumId) {
			log.Printf("Invalid album ID provided: %s", albumId)
			continue
		}

		result, err := s.db.Exec(`
			DELETE FROM starred_albums
			WHERE user_id = $1 AND album_id = $2
		`, userId, albumId)

		if err != nil {
			log.Printf("Error unstarring album %s: %v", albumId, err)
			errors = append(errors, fmt.Sprintf("Failed to unstar album %s", albumId))
		} else {
			if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
				log.Printf("Successfully unstarred album %s for user %d", albumId, userId)
			} else {
				log.Printf("Album %s was not starred by user %d", albumId, userId)
			}
		}
	}

	// Unstar artists
	for _, artistId := range artistIds {
		if !s.isValidID(artistId) {
			log.Printf("Invalid artist ID provided: %s", artistId)
			continue
		}

		result, err := s.db.Exec(`
			DELETE FROM starred_artists
			WHERE user_id = $1 AND artist_id = $2
		`, userId, artistId)

		if err != nil {
			log.Printf("Error unstarring artist %s: %v", artistId, err)
			errors = append(errors, fmt.Sprintf("Failed to unstar artist %s", artistId))
		} else {
			if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
				log.Printf("Successfully unstarred artist %s for user %d", artistId, userId)
			} else {
				log.Printf("Artist %s was not starred by user %d", artistId, userId)
			}
		}
	}

	// If there were critical errors, log them
	if len(errors) > 0 {
		log.Printf("Unstar operation completed with %d errors", len(errors))
	}

	// Send successful response (Subsonic API doesn't send error details for unstar operations)
	s.sendResponse(c, nil)
}

func (s *Service) SetRating(c *gin.Context) { s.sendResponse(c, nil) }

// Scrobble - Registers the local playback of one or more media files
func (s *Service) Scrobble(c *gin.Context) {
	// Get user ID from context
	userId := s.getUserID(c)

	// Get song IDs to scrobble
	songIds := c.QueryArray("id")

	// Get optional time parameter (milliseconds since epoch)
	timeStr := c.Query("time")
	var playedAt time.Time
	if timeStr != "" {
		if timeMs, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
			playedAt = time.Unix(timeMs/1000, 0)
		} else {
			playedAt = time.Now()
		}
	} else {
		playedAt = time.Now()
	}

	// Get optional submission parameter
	submission := c.DefaultQuery("submission", "true") == "true"

	// Only record if submission is true
	if submission {
		for _, songId := range songIds {
			if s.isValidID(songId) {
				// Get song duration
				var duration int
				err := s.db.QueryRow("SELECT duration FROM songs WHERE id = $1", songId).Scan(&duration)
				if err != nil {
					log.Printf("Error getting song duration for scrobble: %v", err)
					continue
				}

				// Record play history
				_, err = s.db.Exec(`
					INSERT INTO play_history (user_id, song_id, played_at, duration_played)
					VALUES ($1, $2, $3, $4)
				`, userId, songId, playedAt, duration)

				if err != nil {
					log.Printf("Error scrobbling song %s: %v", songId, err)
				}
			}
		}
	}

	s.sendResponse(c, nil)
}

// SetNowPlaying - Registers the local playback of a media file
func (s *Service) SetNowPlaying(c *gin.Context) {
	// Get user ID from context
	userId := s.getUserID(c)

	songId := c.Query("id")
	if !s.isValidID(songId) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// Get optional player ID
	playerId := c.DefaultQuery("playerId", "default")

	// Update or insert now playing record
	_, err := s.db.Exec(`
		INSERT INTO now_playing (user_id, song_id, player_id, started_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, player_id)
		DO UPDATE SET song_id = $2, updated_at = NOW()
	`, userId, songId, playerId)

	if err != nil {
		log.Printf("Error setting now playing: %v", err)
		s.sendError(c, 0, "Failed to set now playing")
		return
	}

	s.sendResponse(c, nil)
}

// GetArtistInfo2 - Returns artist information including biography and similar artists
func (s *Service) GetArtistInfo2(c *gin.Context) {
	id := c.Query("id")
	if !s.isValidID(id) {
		s.sendError(c, 10, "Required parameter 'id' is missing or invalid")
		return
	}

	// Get artist from database to verify it exists
	var artistName string
	err := s.db.QueryRow("SELECT name FROM artists WHERE id = $1", id).Scan(&artistName)
	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "Artist not found")
		} else {
			s.sendError(c, 0, "Database error")
		}
		return
	}

	// Create artist info response with basic information
	artistInfo := &ArtistInfo2{
		Biography: "Artist information is not available at this time.",
		// Optional fields - these would typically come from external services like Last.fm
		// MusicBrainzID: "",
		// LastFmUrl: "",
		// SmallImageUrl: "",
		// MediumImageUrl: "",
		// LargeImageUrl: "",
		SimilarArtist: []ArtistID3{}, // Empty array for now
	}

	// Optionally, get similar artists from the same genre
	count := 6 // Default count for similar artists
	if countStr := c.Query("count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 && parsed <= 20 {
			count = parsed
		}
	}

	// Get artists from the same genre as similar artists
	rows, err := s.db.Query(`
		SELECT DISTINCT ar2.id, ar2.name, COUNT(al2.id) as album_count
		FROM artists ar2
		LEFT JOIN albums al2 ON ar2.id = al2.artist_id
		WHERE ar2.id != $1
		AND EXISTS (
			SELECT 1 FROM albums al1 
			JOIN albums al3 ON al1.genre = al3.genre 
			WHERE al1.artist_id = $1 AND al3.artist_id = ar2.id
		)
		GROUP BY ar2.id, ar2.name
		ORDER BY ar2.name
		LIMIT $2
	`, id, count)

	if err == nil {
		defer rows.Close()
		var similarArtists []ArtistID3

		for rows.Next() {
			var artist ArtistID3
			var albumCount int
			err := rows.Scan(&artist.ID, &artist.Name, &albumCount)
			if err == nil {
				artist.AlbumCount = albumCount
				similarArtists = append(similarArtists, artist)
			}
		}
		artistInfo.SimilarArtist = similarArtists
	}

	s.sendResponse(c, artistInfo)
}

// getContentType returns the content type for a given file extension
func (s *Service) getContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".wav":
		return "audio/wav"
	default:
		return "audio/mpeg"
	}
}

// parseIntDefault parses a string to int with a default value
func parseIntDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	if value, err := strconv.Atoi(s); err == nil {
		return value
	}
	return defaultValue
}

// GetSimilarSongs2 returns songs similar to the given song using Last.fm API
func (s *Service) GetSimilarSongs2(c *gin.Context) {
	// Get parameters
	id := c.Query("id")
	if id == "" {
		s.sendError(c, 10, "Required parameter 'id' is missing")
		return
	}

	size := 50 // default size
	if sizeStr := c.Query("count"); sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil && parsed > 0 && parsed <= 500 {
			size = parsed
		}
	}

	// Get song information from database
	var songTitle, artistName string
	var albumId int
	query := `
		SELECT s.title, ar.name as artist_name, s.album_id
		FROM songs s
		JOIN artists ar ON s.artist_id = ar.id
		WHERE s.id = $1`

	err := s.db.QueryRow(query, id).Scan(&songTitle, &artistName, &albumId)
	if err != nil {
		if err == sql.ErrNoRows {
			s.sendError(c, 70, "Song not found")
			return
		}
		s.sendError(c, 0, "Database error")
		return
	}

	var songs []Child

	// Debug logging
	log.Printf("DEBUG: Looking for similar songs to: %s - %s (ID: %s)", artistName, songTitle, id)

	// Try to get similar songs from Last.fm
	if s.lastfm != nil {
		log.Printf("DEBUG: Trying Last.fm API...")
		similarTracks, err := s.lastfm.GetSimilarTracks(artistName, songTitle, size)
		if err == nil && len(similarTracks.Track) > 0 {
			log.Printf("DEBUG: Last.fm returned %d similar tracks", len(similarTracks.Track))
			// Convert Last.fm tracks to local songs
			songs = s.findLocalSongsFromLastFM(similarTracks.Track, size)
			log.Printf("DEBUG: Found %d local matches from Last.fm", len(songs))
		} else {
			log.Printf("DEBUG: Last.fm error or no results: %v", err)
		}
	} else {
		log.Printf("DEBUG: Last.fm service not available")
	}

	// If we don't have enough songs from Last.fm, use fallback strategy
	minSongs := size / 4 // Al menos 25% de las canciones solicitadas
	if minSongs < 1 {
		minSongs = 1 // Asegurar al menos 1 canción como mínimo
	}

	if len(songs) < minSongs {
		log.Printf("DEBUG: Using fallback strategies (current: %d, min needed: %d)", len(songs), minSongs)
		fallbackSongs := s.getFallbackSimilarSongs(artistName, albumId, size-len(songs), id)
		log.Printf("DEBUG: Fallback found %d songs", len(fallbackSongs))
		songs = append(songs, fallbackSongs...)
	}

	log.Printf("DEBUG: Final result: %d songs", len(songs))

	// Limit results to requested size
	if len(songs) > size {
		songs = songs[:size]
	}

	log.Printf("DEBUG: Creating SimilarSongs2 response with %d songs", len(songs))

	result := &SimilarSongs2{
		Song: songs,
	}

	log.Printf("DEBUG: About to send response with %d songs in result.Song", len(result.Song))
	s.sendResponse(c, result)
}

// findLocalSongsFromLastFM tries to find local songs matching Last.fm recommendations
func (s *Service) findLocalSongsFromLastFM(lastfmTracks []lastfm.Track, limit int) []Child {
	var songs []Child

	for _, track := range lastfmTracks {
		if len(songs) >= limit {
			break
		}

		// Try to find the song in our database
		query := `
			SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
			       s.file_size, s.bitrate, s.format, s.album_id,
			       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
			FROM songs s
			JOIN artists ar ON s.artist_id = ar.id
			JOIN albums al ON s.album_id = al.id
			WHERE LOWER(s.title) LIKE LOWER($1) AND LOWER(ar.name) LIKE LOWER($2)
			LIMIT 1`

		var song Child
		var coverArtPath sql.NullString
		var year sql.NullInt32
		var genre sql.NullString

		err := s.db.QueryRow(query, "%"+track.Name+"%", "%"+track.Artist.Name+"%").Scan(
			&song.ID, &song.Title, &song.Track, &song.Duration, &song.Path,
			&song.Size, &song.BitRate, &song.Suffix, &song.AlbumId,
			&song.Artist, &song.Album, &year, &genre, &coverArtPath,
		)

		if err == nil {
			song.IsDir = false
			if year.Valid {
				song.Year = int(year.Int32)
			}
			if genre.Valid {
				song.Genre = genre.String
			}
			if coverArtPath.Valid {
				song.CoverArt = song.AlbumId
			}

			songs = append(songs, song)
		}
	}

	return songs
}

// getFallbackSimilarSongs provides fallback recommendations based on local metadata
func (s *Service) getFallbackSimilarSongs(artistName string, albumId int, limit int, excludeId string) []Child {
	var songs []Child

	// Strategy 1: Songs from the same artist
	query := `
		SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
		       s.file_size, s.bitrate, s.format, s.album_id,
		       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
		FROM songs s
		JOIN artists ar ON s.artist_id = ar.id
		JOIN albums al ON s.album_id = al.id
		WHERE ar.name = $1 AND s.id != $2
		ORDER BY RANDOM()
		LIMIT $3`

	rows, err := s.db.Query(query, artistName, excludeId, limit/2)
	if err == nil {
		songs = append(songs, s.scanSongs(rows)...)
		rows.Close()
	}

	// Strategy 2: Songs from the same album
	if len(songs) < limit {
		query = `
			SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
			       s.file_size, s.bitrate, s.format, s.album_id,
			       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
			FROM songs s
			JOIN artists ar ON s.artist_id = ar.id
			JOIN albums al ON s.album_id = al.id
			WHERE s.album_id = $1 AND s.id != $2
			ORDER BY s.track_number
			LIMIT $3`

		rows, err := s.db.Query(query, albumId, excludeId, limit-len(songs))
		if err == nil {
			songs = append(songs, s.scanSongs(rows)...)
			rows.Close()
		}
	}

	// Strategy 3: Songs from the same genre
	if len(songs) < limit {
		query = `
			SELECT s.id, s.title, s.track_number, s.duration, s.file_path, 
			       s.file_size, s.bitrate, s.format, s.album_id,
			       ar.name as artist_name, al.name as album_name, al.year, al.genre, al.cover_art_path
			FROM songs s
			JOIN artists ar ON s.artist_id = ar.id
			JOIN albums al ON s.album_id = al.id
			WHERE al.genre = (SELECT genre FROM albums WHERE id = $1) 
				AND s.id != $2 AND ar.name != $3
			ORDER BY RANDOM()
			LIMIT $4`

		rows, err := s.db.Query(query, albumId, excludeId, artistName, limit-len(songs))
		if err == nil {
			songs = append(songs, s.scanSongs(rows)...)
			rows.Close()
		}
	}

	return songs
}

// scanSongs is a helper function to scan song results
func (s *Service) scanSongs(rows *sql.Rows) []Child {
	var songs []Child

	for rows.Next() {
		var song Child
		var coverArtPath sql.NullString
		var year sql.NullInt32
		var genre sql.NullString

		err := rows.Scan(
			&song.ID, &song.Title, &song.Track, &song.Duration, &song.Path,
			&song.Size, &song.BitRate, &song.Suffix, &song.AlbumId,
			&song.Artist, &song.Album, &year, &genre, &coverArtPath,
		)

		if err == nil {
			song.IsDir = false
			if year.Valid {
				song.Year = int(year.Int32)
			}
			if genre.Valid {
				song.Genre = genre.String
			}
			if coverArtPath.Valid {
				song.CoverArt = song.AlbumId
			}

			songs = append(songs, song)
		}
	}

	return songs
}
