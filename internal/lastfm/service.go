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
	// API Key gratuita para desarrollo - en producción debería venir del config
	APIKey = "YOUR_LASTFM_API_KEY" // TODO: Mover a configuración
)

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
	Name       string `json:"name"`
	Duration   string `json:"duration,omitempty"`
	PlayCount  string `json:"playcount,omitempty"`
	Listeners  string `json:"listeners,omitempty"`
	Match      string `json:"match"`
	MBID       string `json:"mbid,omitempty"`
	URL        string `json:"url"`
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

func NewService(apiKey string) *Service {
	if apiKey == "" {
		apiKey = APIKey // Fallback a la clave por defecto
	}

	return &Service{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		apiKey: apiKey,
	}
}

// GetSimilarTracks obtiene canciones similares desde Last.fm
func (s *Service) GetSimilarTracks(artist, track string, limit int) (*SimilarTracksResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 30 // Default limit
	}

	log.Printf("DEBUG: Last.fm API Key being used: %s...", s.apiKey[:12]) // Solo mostrar primeros 12 caracteres

	params := url.Values{}
	params.Set("method", "track.getsimilar")
	params.Set("artist", artist)
	params.Set("track", track)
	params.Set("api_key", s.apiKey)
	params.Set("format", "json")
	params.Set("limit", fmt.Sprintf("%d", limit))

	requestURL := fmt.Sprintf("%s?%s", LastFMAPIURL, params.Encode())
	log.Printf("DEBUG: Last.fm request URL: %s", requestURL)

	resp, err := s.client.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("error making request to Last.fm: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("DEBUG: Last.fm response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Last.fm API returned status %d", resp.StatusCode)
	}

	var lastfmResp LastFMResponse
	if err := json.NewDecoder(resp.Body).Decode(&lastfmResp); err != nil {
		return nil, fmt.Errorf("error decoding Last.fm response: %w", err)
	}

	if lastfmResp.Error != 0 {
		return nil, fmt.Errorf("Last.fm API error %d: %s", lastfmResp.Error, lastfmResp.Message)
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
		return nil, fmt.Errorf("error making request to Last.fm: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Last.fm API returned status %d", resp.StatusCode)
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
		return nil, fmt.Errorf("error decoding Last.fm response: %w", err)
	}

	if response.Error != 0 {
		return nil, fmt.Errorf("Last.fm API error %d: %s", response.Error, response.Message)
	}

	// Crear una respuesta con artistas similares (sin tracks específicos)
	result := &SimilarTracksResponse{
		Track: []Track{},
	}

	// Por simplicidad, retornamos los nombres de artistas similares
	// En una implementación más completa, podrías obtener top tracks de cada artista similar
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
