package subsonic

import (
	"database/sql"
	"encoding/xml"
	"log"
	"net/http"

	"castafiore-backend/internal/auth"
	"castafiore-backend/internal/lastfm"

	"github.com/gin-gonic/gin"
)

type Service struct {
	db     *sql.DB
	auth   *auth.Service
	lastfm *lastfm.Service
}

// Response structures for Subsonic API
type SubsonicResponse struct {
	XMLName       xml.Name           `xml:"subsonic-response" json:"-"`
	Status        string             `xml:"status,attr" json:"status"`
	Version       string             `xml:"version,attr" json:"version"`
	Type          string             `xml:"type,attr" json:"type"`
	Error         *Error             `xml:"error,omitempty" json:"error,omitempty"`
	License       *License           `xml:"license,omitempty" json:"license,omitempty"`
	MusicFolders  *MusicFolders      `xml:"musicFolders,omitempty" json:"musicFolders,omitempty"`
	Indexes       *Indexes           `xml:"indexes,omitempty" json:"indexes,omitempty"`
	Directory     *Directory         `xml:"directory,omitempty" json:"directory,omitempty"`
	Genres        *Genres            `xml:"genres,omitempty" json:"genres,omitempty"`
	Artists       *ArtistsID3        `xml:"artists,omitempty" json:"artists,omitempty"`
	Artist        *ArtistWithAlbums  `xml:"artist,omitempty" json:"artist,omitempty"`
	Album         *AlbumID3          `xml:"album,omitempty" json:"album,omitempty"`
	Song          *Child             `xml:"song,omitempty" json:"song,omitempty"`
	SearchResult3 *SearchResult3     `xml:"searchResult3,omitempty" json:"searchResult3,omitempty"`
	TopSongs      *TopSongs          `xml:"topSongs,omitempty" json:"topSongs,omitempty"`
	AlbumList2    *AlbumList2        `xml:"albumList2,omitempty" json:"albumList2,omitempty"`
	RandomSongs   *RandomSongs       `xml:"randomSongs,omitempty" json:"randomSongs,omitempty"`
	SongsByGenre  *SongsByGenre      `xml:"songsByGenre,omitempty" json:"songsByGenre,omitempty"`
	SimilarSongs2 *SimilarSongs2     `xml:"similarSongs2,omitempty" json:"similarSongs2,omitempty"`
	NowPlaying    *NowPlaying        `xml:"nowPlaying,omitempty" json:"nowPlaying,omitempty"`
	Starred       *Starred           `xml:"starred,omitempty" json:"starred,omitempty"`
	Starred2      *Starred2          `xml:"starred2,omitempty" json:"starred2,omitempty"`
	ArtistInfo2   *ArtistInfo2       `xml:"artistInfo2,omitempty" json:"artistInfo2,omitempty"`
	Playlists     *Playlists         `xml:"playlists,omitempty" json:"playlists,omitempty"`
	Playlist      *PlaylistWithSongs `xml:"playlist,omitempty" json:"playlist,omitempty"`
	User          *User              `xml:"user,omitempty" json:"user,omitempty"`
}

type Error struct {
	Code    int    `xml:"code,attr" json:"code"`
	Message string `xml:"message,attr" json:"message"`
}

type License struct {
	Valid bool `xml:"valid,attr" json:"valid"`
}

type MusicFolders struct {
	MusicFolder []MusicFolder `xml:"musicFolder" json:"musicFolder"`
}

type MusicFolder struct {
	ID   int    `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type Indexes struct {
	LastModified int64   `xml:"lastModified,attr" json:"lastModified"`
	Index        []Index `xml:"index" json:"index"`
}

type Index struct {
	Name   string   `xml:"name,attr" json:"name"`
	Artist []Artist `xml:"artist" json:"artist"`
}

type Artist struct {
	ID   string `xml:"id,attr" json:"id"`
	Name string `xml:"name,attr" json:"name"`
}

type Directory struct {
	ID    string  `xml:"id,attr" json:"id"`
	Name  string  `xml:"name,attr" json:"name"`
	Child []Child `xml:"child" json:"child"`
}

type Child struct {
	ID          string `xml:"id,attr" json:"id"`
	Parent      string `xml:"parent,attr,omitempty" json:"parent,omitempty"`
	AlbumId     string `xml:"albumId,attr,omitempty" json:"albumId"`
	IsDir       bool   `xml:"isDir,attr" json:"isDir"`
	Title       string `xml:"title,attr" json:"title"`
	Album       string `xml:"album,attr,omitempty" json:"album,omitempty"`
	Artist      string `xml:"artist,attr,omitempty" json:"artist,omitempty"`
	Track       int    `xml:"track,attr" json:"track"`
	Year        int    `xml:"year,attr" json:"year"`
	Genre       string `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	CoverArt    string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Size        int64  `xml:"size,attr,omitempty" json:"size,omitempty"`
	ContentType string `xml:"contentType,attr,omitempty" json:"contentType,omitempty"`
	Suffix      string `xml:"suffix,attr,omitempty" json:"suffix,omitempty"`
	Duration    int    `xml:"duration,attr" json:"duration"`
	BitRate     int    `xml:"bitRate,attr" json:"bitRate"`
	Path        string `xml:"path,attr,omitempty" json:"path,omitempty"`
}

type Genres struct {
	Genre []Genre `xml:"genre" json:"genre"`
}

type Genre struct {
	SongCount  int    `xml:"songCount,attr" json:"songCount"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
	Value      string `xml:",chardata" json:"value"`
}

type ArtistsID3 struct {
	Index []IndexID3 `xml:"index" json:"index"`
}

type IndexID3 struct {
	Name   string      `xml:"name,attr" json:"name"`
	Artist []ArtistID3 `xml:"artist" json:"artist"`
}

type ArtistID3 struct {
	ID         string `xml:"id,attr" json:"id"`
	Name       string `xml:"name,attr" json:"name"`
	CoverArt   string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	AlbumCount int    `xml:"albumCount,attr" json:"albumCount"`
	Starred    string `xml:"starred,attr,omitempty" json:"starred,omitempty"`
}

type AlbumID3 struct {
	ID        string  `xml:"id,attr" json:"id"`
	Name      string  `xml:"name,attr" json:"name"`
	Artist    string  `xml:"artist,attr" json:"artist"`
	ArtistID  string  `xml:"artistId,attr" json:"artistId"`
	CoverArt  string  `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	SongCount int     `xml:"songCount,attr" json:"songCount"`
	Duration  int     `xml:"duration,attr" json:"duration"`
	Created   string  `xml:"created,attr" json:"created"`
	Year      int     `xml:"year,attr,omitempty" json:"year,omitempty"`
	Genre     string  `xml:"genre,attr,omitempty" json:"genre,omitempty"`
	Song      []Child `xml:"song,omitempty" json:"song,omitempty"`
}

type SearchResult3 struct {
	Artist []ArtistID3 `xml:"artist" json:"artist"`
	Album  []AlbumID3  `xml:"album" json:"album"`
	Song   []Child     `xml:"song" json:"song"`
}

type TopSongs struct {
	Song []Child `xml:"song" json:"song"`
}

type RandomSongs struct {
	Song []Child `xml:"song" json:"song"`
}

type SongsByGenre struct {
	Song []Child `xml:"song" json:"song"`
}

type SimilarSongs2 struct {
	Song []Child `xml:"song" json:"song"`
}

type NowPlaying struct {
	Entry []NowPlayingEntry `xml:"entry" json:"entry"`
}

type NowPlayingEntry struct {
	Child
	Username   string `xml:"username,attr" json:"username"`
	MinutesAgo int    `xml:"minutesAgo,attr" json:"minutesAgo"`
	PlayerId   string `xml:"playerId,attr" json:"playerId"`
}

type Starred struct {
	Artist []Artist `xml:"artist" json:"artist"`
	Album  []Child  `xml:"album" json:"album"`
	Song   []Child  `xml:"song" json:"song"`
}

type Starred2 struct {
	Artist []ArtistID3 `xml:"artist" json:"artist"`
	Album  []AlbumID3  `xml:"album" json:"album"`
	Song   []Child     `xml:"song" json:"song"`
}

type AlbumList2 struct {
	Album []AlbumID3 `xml:"album" json:"album"`
}

type ArtistWithAlbums struct {
	ID         string     `xml:"id,attr" json:"id"`
	Name       string     `xml:"name,attr" json:"name"`
	AlbumCount int        `xml:"albumCount,attr" json:"albumCount"`
	Album      []AlbumID3 `xml:"album" json:"album"`
}

type ArtistInfo2 struct {
	Biography      string      `xml:"biography,omitempty" json:"biography,omitempty"`
	MusicBrainzID  string      `xml:"musicBrainzId,omitempty" json:"musicBrainzId,omitempty"`
	LastFmUrl      string      `xml:"lastFmUrl,omitempty" json:"lastFmUrl,omitempty"`
	SmallImageUrl  string      `xml:"smallImageUrl,omitempty" json:"smallImageUrl,omitempty"`
	MediumImageUrl string      `xml:"mediumImageUrl,omitempty" json:"mediumImageUrl,omitempty"`
	LargeImageUrl  string      `xml:"largeImageUrl,omitempty" json:"largeImageUrl,omitempty"`
	SimilarArtist  []ArtistID3 `xml:"similarArtist,omitempty" json:"similarArtist,omitempty"`
}

type Playlists struct {
	Playlist []PlaylistID3 `xml:"playlist" json:"playlist"`
}

type PlaylistID3 struct {
	ID        string `xml:"id,attr" json:"id"`
	Name      string `xml:"name,attr" json:"name"`
	Comment   string `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Owner     string `xml:"owner,attr" json:"owner"`
	Public    bool   `xml:"public,attr" json:"public"`
	SongCount int    `xml:"songCount,attr" json:"songCount"`
	Duration  int    `xml:"duration,attr" json:"duration"`
	Created   string `xml:"created,attr" json:"created"`
	Changed   string `xml:"changed,attr" json:"changed"`
	CoverArt  string `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
}

type PlaylistWithSongs struct {
	ID        string  `xml:"id,attr" json:"id"`
	Name      string  `xml:"name,attr" json:"name"`
	Comment   string  `xml:"comment,attr,omitempty" json:"comment,omitempty"`
	Owner     string  `xml:"owner,attr" json:"owner"`
	Public    bool    `xml:"public,attr" json:"public"`
	SongCount int     `xml:"songCount,attr" json:"songCount"`
	Duration  int     `xml:"duration,attr" json:"duration"`
	Created   string  `xml:"created,attr" json:"created"`
	Changed   string  `xml:"changed,attr" json:"changed"`
	CoverArt  string  `xml:"coverArt,attr,omitempty" json:"coverArt,omitempty"`
	Entry     []Child `xml:"entry" json:"entry"`
}

type User struct {
	Username            string `xml:"username,attr" json:"username"`
	Email               string `xml:"email,attr,omitempty" json:"email,omitempty"`
	ScrobblingEnabled   bool   `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	AdminRole           bool   `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole        bool   `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole        bool   `xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole          bool   `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole        bool   `xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole        bool   `xml:"coverArtRole,attr" json:"coverArtRole"`
	CommentRole         bool   `xml:"commentRole,attr" json:"commentRole"`
	PodcastRole         bool   `xml:"podcastRole,attr" json:"podcastRole"`
	StreamRole          bool   `xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole         bool   `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	ShareRole           bool   `xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool   `xml:"videoConversionRole,attr" json:"videoConversionRole"`
	MaxBitRate          int    `xml:"maxBitRate,attr,omitempty" json:"maxBitRate,omitempty"`
}

func NewService(db *sql.DB, authService *auth.Service, lastfmAPIKey string) *Service {
	return &Service{
		db:     db,
		auth:   authService,
		lastfm: lastfm.NewService(lastfmAPIKey),
	}
}

// AuthMiddleware handles authentication for Subsonic API requests
func (s *Service) AuthMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Para simplificar, permitir todas las peticiones por ahora
		// TODO: Implementar autenticación Subsonic real (u/p, t/s parameters)
		c.Next()
	})
}

func (s *Service) sendResponse(c *gin.Context, data interface{}) {
	format := c.DefaultQuery("f", "xml")

	response := SubsonicResponse{
		Status:  "ok",
		Version: "1.16.1",
		Type:    "castafiore",
	}

	// Set the data field based on type
	switch v := data.(type) {
	case *License:
		response.License = v
	case *MusicFolders:
		response.MusicFolders = v
	case *Indexes:
		response.Indexes = v
	case *Directory:
		response.Directory = v
	case *Genres:
		response.Genres = v
	case *ArtistsID3:
		response.Artists = v
	case *ArtistID3:
		// Este caso no se usa, ArtistID3 va dentro de ArtistsID3
	case *ArtistWithAlbums:
		response.Artist = v
	case *AlbumID3:
		response.Album = v
	case *Child:
		response.Song = v
	case *SearchResult3:
		response.SearchResult3 = v
	case *AlbumList2:
		response.AlbumList2 = v
	case *TopSongs:
		response.TopSongs = v
	case *RandomSongs:
		response.RandomSongs = v
	case *SongsByGenre:
		response.SongsByGenre = v
	case *SimilarSongs2:
		log.Printf("DEBUG: Setting SimilarSongs2 with %d songs", len(v.Song))
		response.SimilarSongs2 = v
	case *NowPlaying:
		response.NowPlaying = v
	case *Starred:
		response.Starred = v
	case *Starred2:
		response.Starred2 = v
	case *ArtistInfo2:
		response.ArtistInfo2 = v
	case *Playlists:
		response.Playlists = v
	case *PlaylistWithSongs:
		response.Playlist = v
	case *User:
		response.User = v
	}

	if format == "json" {
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, gin.H{"subsonic-response": response})
	} else {
		c.Header("Content-Type", "text/xml")
		c.XML(http.StatusOK, response)
	}
}

func (s *Service) sendError(c *gin.Context, code int, message string) {
	format := c.DefaultQuery("f", "xml")

	response := SubsonicResponse{
		Status:  "failed",
		Version: "1.16.1",
		Type:    "castafiore",
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}

	if format == "json" {
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, gin.H{"subsonic-response": response})
	} else {
		c.Header("Content-Type", "text/xml")
		c.XML(http.StatusOK, response)
	}
	c.Abort()
}

// isValidID checks if the provided ID is valid (not empty, not "undefined", not "null")
func (s *Service) isValidID(id string) bool {
	if id == "" || id == "undefined" || id == "null" {
		return false
	}
	return true
}

// getUserID extracts user ID from the authenticated context
// For now returns a fallback user ID of 1, but should be updated to use proper authentication
func (s *Service) getUserID(c *gin.Context) int {
	// TODO: Extract from JWT token or Subsonic authentication in the context
	// For now, return default user ID 1
	// This should be set by the AuthMiddleware after successful authentication
	if userID, exists := c.Get("userID"); exists {
		if id, ok := userID.(int); ok {
			return id
		}
	}

	// Fallback to user ID 1 for now
	return 1
}
