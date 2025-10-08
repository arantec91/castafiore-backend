package lastfm

import (
	"encoding/json"
	"fmt"
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
	client *http.Client
	apiKey string
}

type LastFMResponse struct {
	SimilarTracks *SimilarTracksResponse `json:"similartracks,omitempty"`
	Error         int                    `json:"error,omitempty"`
	Message       string                 `json:"message,omitempty"`
}

type SimilarTracksResponse struct {
	Track []Track `json:"track"`
	Attr  struct {
		Artist string `json:"artist"`
	} `json:"@attr"`
}

type Track struct {
	Name       string  `json:"name"`
	Duration   int     `json:"duration,omitempty"`
	PlayCount  int     `json:"playcount,omitempty"`
	Listeners  int     `json:"listeners,omitempty"`
	Match      float64 `json:"match"`
	MBID       string  `json:"mbid,omitempty"`
	URL        string  `json:"url"`
	Streamable struct {
		Text      string `json:"#text"`
		Fulltrack string `json:"fulltrack"`
	} `json:"streamable"`
	Artist struct {
		Name string `json:"name"`
		MBID string `json:"mbid,omitempty"`
		URL  string `json:"url"`
	} `json:"artist"`
	Images []struct {
		Text string `json:"#text"`
		Size string `json:"size"`
	} `json:"image"`
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
		apiKey: config.APIKey,
	}
}

// GetSimilarTracks obtiene canciones similares desde Last.fm
func (s *Service) GetSimilarTracks(artist, track string, limit int) (*SimilarTracksResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 30 // Default limit
	}

	log.Printf("DEBUG: last.fm API key being used: %s...", s.apiKey[:12])

	params := url.Values{}
	params.Set("method", "track.getsimilar")
	params.Set("artist", artist)
	params.Set("track", track)
	params.Set("api_key", s.apiKey)
	params.Set("format", "json")
	params.Set("limit", fmt.Sprintf("%d", limit))

	requestURL := fmt.Sprintf("%s?%s", LastFMAPIURL, params.Encode())
	log.Printf("DEBUG: last.fm request URL: %s", requestURL)

	resp, err := s.client.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("error making request to last.fm: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("DEBUG: last.fm response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("last.fm API returned status %d", resp.StatusCode)
	}

	var lastfmResp LastFMResponse
	if err := json.NewDecoder(resp.Body).Decode(&lastfmResp); err != nil {
		return nil, fmt.Errorf("error decoding last.fm response: %w", err)
	}

	if lastfmResp.Error != 0 {
		return nil, fmt.Errorf("last.fm API error %d: %s", lastfmResp.Error, lastfmResp.Message)
	}

	if lastfmResp.SimilarTracks == nil {
		return &SimilarTracksResponse{Track: []Track{}}, nil
	}

	return lastfmResp.SimilarTracks, nil
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
		Track: []Track{},
	}

	// Por simplicidad, retornamos los nombres de artistas similares
	for _, artist := range response.SimilarArtists.Artist {
		track := Track{
			Name: "", // No tenemos track específico
			Artist: struct {
				Name string `json:"name"`
				MBID string `json:"mbid,omitempty"`
				URL  string `json:"url"`
			}{
				Name: artist.Name,
			},
		}
		result.Track = append(result.Track, track)

		if len(result.Track) >= limit {
			break
		}
	}

	return result, nil
}
