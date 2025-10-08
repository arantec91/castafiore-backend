package lastfm

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	LastFMAPIURL = "https://ws.audioscrobbler.com/2.0/"
)

// Config holds Last.fm configuration
type Config struct {
	APIKey string
}

type Service struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

// Streamable represents the Last.fm streamable object which can be either a string or an object
type Streamable struct {
	Text      string `json:"#text,omitempty"`
	FullTrack string `json:"fulltrack,omitempty"`
}

// UnmarshalJSON implements custom unmarshaling for Streamable
func (s *Streamable) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		s.Text = str
		return nil
	}

	// If that fails, try to unmarshal as object
	type streamableObj struct {
		Text      string `json:"#text,omitempty"`
		FullTrack string `json:"fulltrack,omitempty"`
	}
	var obj streamableObj
	if err := json.Unmarshal(data, &obj); err == nil {
		s.Text = obj.Text
		s.FullTrack = obj.FullTrack
		return nil
	}

	// If both fail, return the object unmarshal error
	return fmt.Errorf("unable to unmarshal streamable field")
}

// Common fields for all track types
type BaseTrack struct {
	Name       string     `json:"name"`
	Duration   int        `json:"duration,omitempty"`
	MBID       string     `json:"mbid,omitempty"`
	URL        string     `json:"url"`
	Streamable Streamable `json:"streamable"`
	Artist     struct {
		Name string `json:"name"`
		MBID string `json:"mbid,omitempty"`
		URL  string `json:"url"`
	} `json:"artist"`
	Images []struct {
		Text string `json:"#text"`
		Size string `json:"size"`
	} `json:"image"`
}

// TopTrack is used for artist.getTopTracks responses where playcount/listeners are strings
type TopTrack struct {
	BaseTrack
	PlayCount string `json:"playcount,omitempty"`
	Listeners string `json:"listeners,omitempty"`
}

// SimilarTrack is used for track.getSimilar responses where playcount/listeners are numbers
type SimilarTrack struct {
	BaseTrack
	PlayCount int     `json:"playcount,omitempty"`
	Listeners int     `json:"listeners,omitempty"`
	Match     float64 `json:"match"`
}

type SimilarTracksResponse struct {
	Track []SimilarTrack `json:"track"`
	Attr  struct {
		Artist string `json:"artist"`
	} `json:"@attr"`
}

// TopTracksResponse represents the response from Last.fm's artist.getTopTracks method
type TopTracksResponse struct {
	Error     int    `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
	Toptracks struct {
		Track []TopTrack `json:"track"`
		Attr  struct {
			Artist string `json:"artist"`
			Page   string `json:"page"`
			Total  string `json:"total"`
		} `json:"@attr"`
	} `json:"toptracks"`
}

type LastFMResponse struct {
	SimilarTracks *SimilarTracksResponse `json:"similartracks,omitempty"`
	Error         int                    `json:"error,omitempty"`
	Message       string                 `json:"message,omitempty"`
}

func NewService(config Config) *Service {
	log.Printf("Initializing Last.fm service with API key: %s...", config.APIKey[:12])
	if config.APIKey == "" {
		log.Printf("Warning: Last.fm API key is empty")
		return nil
	}

	return &Service{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiKey:  config.APIKey,
		baseURL: "http://ws.audioscrobbler.com/2.0/",
	}
}

// GetSimilarTracks fetches similar tracks for the given track from Last.fm
func (s *Service) GetSimilarTracks(artistName, trackName string) (*SimilarTracksResponse, error) {
	// Build URL with parameters
	params := url.Values{}
	params.Add("method", "track.getSimilar")
	params.Add("artist", artistName)
	params.Add("track", trackName)
	params.Add("api_key", s.apiKey)
	params.Add("format", "json")

	resp, err := s.client.Get(fmt.Sprintf("%s?%s", s.baseURL, params.Encode()))
	if err != nil {
		log.Printf("[LastFM] Error fetching similar tracks for %s - %s: %v", artistName, trackName, err)
		return nil, fmt.Errorf("error fetching similar tracks: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[LastFM] Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Log raw response for debugging
	log.Printf("[LastFM] Raw response: %s", string(body))

	var lfmResp LastFMResponse
	if err := json.Unmarshal(body, &lfmResp); err != nil {
		log.Printf("[LastFM] Error unmarshalling response: %v\nResponse body: %s", err, string(body))
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if lfmResp.Error != 0 {
		log.Printf("[LastFM] API returned error %d: %s", lfmResp.Error, lfmResp.Message)
		return nil, fmt.Errorf("last.fm API error: %s", lfmResp.Message)
	}

	if lfmResp.SimilarTracks == nil {
		log.Printf("[LastFM] No similar tracks found for %s - %s", artistName, trackName)
		return &SimilarTracksResponse{Track: []SimilarTrack{}}, nil
	}

	// Add the artist name to the response attributes if not present
	if lfmResp.SimilarTracks.Attr.Artist == "" {
		lfmResp.SimilarTracks.Attr.Artist = artistName
	}

	// Initialize an empty slice if no tracks were found
	if lfmResp.SimilarTracks.Track == nil {
		lfmResp.SimilarTracks.Track = []SimilarTrack{}
	}

	log.Printf("[LastFM] Successfully fetched %d similar tracks for %s - %s",
		len(lfmResp.SimilarTracks.Track), artistName, trackName)

	return lfmResp.SimilarTracks, nil
}

// GetSimilarTracksByArtist obtiene artistas similares y luego canciones populares de esos artistas
func (s *Service) GetSimilarTracksByArtist(artist string, limit int) (*SimilarTracksResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	params := url.Values{}
	params.Set("method", "artist.getsimilar")
	params.Set("artist", artist)
	params.Set("api_key", s.apiKey)
	params.Set("format", "json")
	params.Set("limit", "10") // Obtenemos algunos artistas similares

	requestURL := fmt.Sprintf("%s?%s", LastFMAPIURL, params.Encode())

	resp, err := s.client.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("error making request to last.fm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("last.fm API returned status %d", resp.StatusCode)
	}

	var response struct {
		SimilarArtists struct {
			Artist []struct {
				Name string `json:"name"`
			} `json:"artist"`
		} `json:"similarartists"`
		Error   int    `json:"error,omitempty"`
		Message string `json:"message,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding last.fm response: %w", err)
	}

	if response.Error != 0 {
		return nil, fmt.Errorf("last.fm API error %d: %s", response.Error, response.Message)
	}

	result := &SimilarTracksResponse{
		Track: []SimilarTrack{},
	}

	// Por simplicidad, retornamos los nombres de artistas similares
	for _, artist := range response.SimilarArtists.Artist {
		track := SimilarTrack{
			BaseTrack: BaseTrack{
				Name: "", // No tenemos track específico
				Artist: struct {
					Name string `json:"name"`
					MBID string `json:"mbid,omitempty"`
					URL  string `json:"url"`
				}{
					Name: artist.Name,
				},
			},
		}
		result.Track = append(result.Track, track)

		if len(result.Track) >= limit {
			break
		}
	}

	return result, nil
}

// GetTopTracks fetches the top tracks for an artist from Last.fm
func (s *Service) GetTopTracks(artistName string) ([]TopTrack, error) {
	// Build URL with parameters
	params := url.Values{}
	params.Add("method", "artist.getTopTracks")
	params.Add("artist", artistName)
	params.Add("api_key", s.apiKey)
	params.Add("format", "json")

	resp, err := s.client.Get(fmt.Sprintf("%s?%s", s.baseURL, params.Encode()))
	if err != nil {
		log.Printf("[LastFM] Error fetching top tracks for artist %s: %v", artistName, err)
		return nil, fmt.Errorf("error fetching top tracks: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[LastFM] Error reading response body: %v", err)
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var topTracksResp TopTracksResponse
	if err := json.Unmarshal(body, &topTracksResp); err != nil {
		log.Printf("[LastFM] Error unmarshalling response: %v\nResponse body: %s", err, string(body))
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	if topTracksResp.Error != 0 {
		log.Printf("[LastFM] API returned error %d: %s", topTracksResp.Error, topTracksResp.Message)
		return nil, fmt.Errorf("last.fm API error: %s", topTracksResp.Message)
	}

	log.Printf("[LastFM] Successfully fetched %d top tracks for artist %s",
		len(topTracksResp.Toptracks.Track), artistName)
	return topTracksResp.Toptracks.Track, nil
}
