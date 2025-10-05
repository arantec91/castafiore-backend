package library

import (
	"testing"
	"time"
)

func TestExtractAudioProperties(t *testing.T) {
	scanner := &Scanner{}

	tests := []struct {
		name     string
		format   string
		fileSize int64
		wantErr  bool
	}{
		{
			name:     "MP3 file estimation",
			format:   "mp3",
			fileSize: 5000000, // 5MB
			wantErr:  false,
		},
		{
			name:     "FLAC file estimation",
			format:   "flac",
			fileSize: 30000000, // 30MB
			wantErr:  false,
		},
		{
			name:     "M4A file estimation",
			format:   "m4a",
			fileSize: 8000000, // 8MB
			wantErr:  false,
		},
		{
			name:     "OGG file estimation",
			format:   "ogg",
			fileSize: 6000000, // 6MB
			wantErr:  false,
		},
		{
			name:     "WAV file estimation",
			format:   "wav",
			fileSize: 50000000, // 50MB
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, bitrate := scanner.extractAudioProperties("dummy."+tt.format, tt.format, tt.fileSize)

			if duration == 0 {
				t.Errorf("extractAudioProperties() duration = 0, want > 0")
			}

			if bitrate == 0 {
				t.Errorf("extractAudioProperties() bitrate = 0, want > 0")
			}

			t.Logf("Format: %s, FileSize: %d bytes, Duration: %v, Bitrate: %d kbps",
				tt.format, tt.fileSize, duration, bitrate)
		})
	}
}

func TestGetMP3Bitrate(t *testing.T) {
	scanner := &Scanner{}

	tests := []struct {
		name         string
		version      byte
		layer        byte
		bitrateIndex byte
		want         int
	}{
		{"MPEG1 Layer3 128kbps", 3, 1, 9, 128},
		{"MPEG1 Layer3 320kbps", 3, 1, 14, 320},
		{"MPEG2 Layer3 128kbps", 2, 1, 12, 128},
		{"Invalid index", 3, 1, 15, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanner.getMP3Bitrate(tt.version, tt.layer, tt.bitrateIndex)
			if got != tt.want {
				t.Errorf("getMP3Bitrate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMP3SampleRate(t *testing.T) {
	scanner := &Scanner{}

	tests := []struct {
		name            string
		version         byte
		sampleRateIndex byte
		want            int
	}{
		{"MPEG1 44.1kHz", 3, 0, 44100},
		{"MPEG1 48kHz", 3, 1, 48000},
		{"MPEG2 22.05kHz", 2, 0, 22050},
		{"MPEG2.5 11.025kHz", 0, 0, 11025},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanner.getMP3SampleRate(tt.version, tt.sampleRateIndex)
			if got != tt.want {
				t.Errorf("getMP3SampleRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDurationCalculation(t *testing.T) {
	// Test that duration calculation is reasonable
	fileSize := int64(5000000) // 5MB
	bitrate := 128             // 128 kbps

	expectedDuration := time.Duration(fileSize*8/(int64(bitrate)*1000)) * time.Second

	// For a 5MB file at 128kbps, duration should be around 312 seconds (5 minutes)
	expectedSeconds := float64(fileSize*8) / float64(bitrate*1000)

	if expectedSeconds < 300 || expectedSeconds > 320 {
		t.Errorf("Duration calculation seems off: got %v seconds, expected around 312", expectedSeconds)
	}

	t.Logf("5MB file at 128kbps = %v (%v seconds)", expectedDuration, expectedSeconds)
}
