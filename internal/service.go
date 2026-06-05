// Package internal wires together the Spotify provider, the gRPC/HTTP service
// layer, and the MCP server.
package internal

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Kavin-bm/Spotify-MCP/config"
	"github.com/Kavin-bm/Spotify-MCP/types"
)

// Service holds business logic and delegates to the SpotifyProvider.
// It also handles smart device resolution so callers never need to pass
// a device_id manually.
type Service struct {
	provider types.SpotifyProvider

	// deviceName is the configured Spotify Connect device name (e.g. librespot --name).
	deviceName string
	// configuredDeviceID is an explicit device ID that bypasses ListDevices entirely.
	configuredDeviceID string

	deviceMu sync.RWMutex
	deviceID string // in-memory cache when redis is unavailable
}

// NewService creates a Service wired to the given provider.
func NewService(provider types.SpotifyProvider, cfg *config.Config) *Service {
	return &Service{
		provider:           provider,
		deviceName:         cfg.DeviceName,
		configuredDeviceID: cfg.DeviceID,
	}
}

// resolveDeviceID returns the device ID to use for a playback call.
// Priority: explicit requestedID > SPOTIFY_DEVICE_ID env > in-memory cache > ListDevices.
func (s *Service) resolveDeviceID(ctx context.Context, requestedID string) (string, error) {
	if requestedID != "" {
		return requestedID, nil
	}
	if s.configuredDeviceID != "" {
		return s.configuredDeviceID, nil
	}
	if s.deviceName == "" {
		return "", nil
	}

	s.deviceMu.RLock()
	cached := s.deviceID
	s.deviceMu.RUnlock()
	if cached != "" {
		return cached, nil
	}

	s.deviceMu.Lock()
	defer s.deviceMu.Unlock()
	if s.deviceID != "" {
		return s.deviceID, nil
	}

	devices, err := s.provider.ListDevices(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list devices while resolving %q: %w", s.deviceName, err)
	}
	for _, d := range devices {
		if d.Name == s.deviceName {
			s.deviceID = d.ID
			return d.ID, nil
		}
	}
	return "", fmt.Errorf("device %q not found — is the Spotify Connect player running?", s.deviceName)
}

func (s *Service) invalidateDeviceCache() {
	s.deviceMu.Lock()
	s.deviceID = ""
	s.deviceMu.Unlock()
}

func isDeviceError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "device") || strings.Contains(msg, "no active")
}

// ── Search ────────────────────────────────────────────────────────────────────

func (s *Service) Search(ctx context.Context, query string, searchTypes []string, limit int) (*types.SearchResult, error) {
	if len(searchTypes) == 0 {
		searchTypes = []string{"track", "album", "artist", "playlist"}
	}
	if limit <= 0 {
		limit = 10
	}
	return s.provider.Search(ctx, query, searchTypes, limit)
}

// ── Playback state ────────────────────────────────────────────────────────────

func (s *Service) NowPlaying(ctx context.Context) (*types.PlaybackState, error) {
	return s.provider.NowPlaying(ctx)
}

// ── Playback control ──────────────────────────────────────────────────────────

func (s *Service) PlayMusic(ctx context.Context, uri, requestedDeviceID string) error {
	deviceID, err := s.resolveDeviceID(ctx, requestedDeviceID)
	if err != nil {
		return err
	}
	if err := s.provider.PlayMusic(ctx, uri, deviceID); err != nil {
		if isDeviceError(err) {
			s.invalidateDeviceCache()
		}
		return err
	}
	return nil
}

func (s *Service) PausePlayback(ctx context.Context, requestedDeviceID string) error {
	deviceID, err := s.resolveDeviceID(ctx, requestedDeviceID)
	if err != nil {
		return err
	}
	return s.provider.PausePlayback(ctx, deviceID)
}

func (s *Service) ResumePlayback(ctx context.Context, requestedDeviceID string) error {
	deviceID, err := s.resolveDeviceID(ctx, requestedDeviceID)
	if err != nil {
		return err
	}
	return s.provider.ResumePlayback(ctx, deviceID)
}

func (s *Service) SkipNext(ctx context.Context, requestedDeviceID string) error {
	deviceID, err := s.resolveDeviceID(ctx, requestedDeviceID)
	if err != nil {
		return err
	}
	return s.provider.SkipToNext(ctx, deviceID)
}

func (s *Service) SkipPrevious(ctx context.Context, requestedDeviceID string) error {
	deviceID, err := s.resolveDeviceID(ctx, requestedDeviceID)
	if err != nil {
		return err
	}
	return s.provider.SkipToPrevious(ctx, deviceID)
}

func (s *Service) SetVolume(ctx context.Context, percent int, requestedDeviceID string) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("volume_percent must be in range [0, 100], got %d", percent)
	}
	deviceID, err := s.resolveDeviceID(ctx, requestedDeviceID)
	if err != nil {
		return err
	}
	return s.provider.SetVolume(ctx, percent, deviceID)
}

func (s *Service) TransferPlayback(ctx context.Context, deviceID string, play bool) error {
	if deviceID == "" {
		return fmt.Errorf("device_id is required — use list_devices to find a device ID")
	}
	if err := s.provider.TransferPlayback(ctx, deviceID, play); err != nil {
		if isDeviceError(err) {
			s.invalidateDeviceCache()
		}
		return err
	}
	return nil
}

// ── Devices ───────────────────────────────────────────────────────────────────

func (s *Service) ListDevices(ctx context.Context) ([]*types.Device, error) {
	return s.provider.ListDevices(ctx)
}

// ── Playlists ─────────────────────────────────────────────────────────────────

func (s *Service) ListPlaylists(ctx context.Context, limit int) ([]*types.Playlist, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.provider.ListPlaylists(ctx, limit)
}

func (s *Service) GetPlaylist(ctx context.Context, playlistID string) (*types.Playlist, error) {
	return s.provider.GetPlaylist(ctx, playlistID)
}

func (s *Service) GetPlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]*types.Track, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.provider.GetPlaylistTracks(ctx, playlistID, limit, offset)
}

func (s *Service) CreatePlaylist(ctx context.Context, title, description string, public bool, trackURIs []string) (*types.Playlist, error) {
	if title == "" {
		title = "My Playlist"
	}
	playlist, err := s.provider.CreatePlaylist(ctx, title, description, public)
	if err != nil {
		return nil, err
	}
	if len(trackURIs) > 0 {
		if err := s.provider.AppendPlaylistTracks(ctx, playlist.ID, trackURIs); err != nil {
			return nil, fmt.Errorf("playlist created but failed to add tracks: %w", err)
		}
	}
	return playlist, nil
}

func (s *Service) AppendPlaylistTracks(ctx context.Context, playlistID string, uris []string) error {
	return s.provider.AppendPlaylistTracks(ctx, playlistID, uris)
}

// ── Library ───────────────────────────────────────────────────────────────────

func (s *Service) ListRecentlyPlayed(ctx context.Context, limit int) ([]*types.Track, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.provider.ListRecentlyPlayed(ctx, limit)
}

func (s *Service) ListSavedTracks(ctx context.Context, limit, offset int) ([]*types.Track, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.provider.ListSavedTracks(ctx, limit, offset)
}

// SearchByType is a convenience wrapper that searches a single content type.
func (s *Service) SearchByType(ctx context.Context, query, contentType string, limit int) (*types.SearchResult, error) {
	return s.Search(ctx, query, []string{contentType}, limit)
}
