// Package config loads application configuration from environment variables
// and an optional .env file. All Spotify OAuth credentials are read from env
// to keep secrets out of the codebase.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the full application configuration.
type Config struct {
	// Spotify OAuth2 credentials
	ClientID     string
	ClientSecret string
	RefreshToken string

	// Optional: target a specific Spotify Connect device by name.
	// When set, the service resolves the device ID automatically and injects
	// it into every playback call — callers never need to supply device_id.
	DeviceName string

	// Optional: explicit Spotify device ID. Takes priority over DeviceName
	// and skips the ListDevices API call entirely.
	DeviceID string

	// Port for the MCP HTTP server (default 8080).
	Port int
}

// Load reads configuration from environment variables.
// Required: SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET, SPOTIFY_REFRESH_TOKEN
func Load() (*Config, error) {
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	refreshToken := os.Getenv("SPOTIFY_REFRESH_TOKEN")

	if clientID == "" || clientSecret == "" || refreshToken == "" {
		return nil, fmt.Errorf(
			"missing required env vars: SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET, SPOTIFY_REFRESH_TOKEN",
		)
	}

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT value %q: %w", p, err)
		}
		port = parsed
	}

	return &Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
		DeviceName:   os.Getenv("SPOTIFY_DEVICE_NAME"),
		DeviceID:     os.Getenv("SPOTIFY_DEVICE_ID"),
		Port:         port,
	}, nil
}
