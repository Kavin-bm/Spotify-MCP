// Package spotify implements the SpotifyProvider interface by wrapping
// the official Spotify Web API using OAuth2 refresh-token authentication.
// Tokens are refreshed automatically — no manual rotation is needed.
package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/Kavin-bm/Spotify-MCP/types"
)

const baseURL = "https://api.spotify.com/v1"

// Handler implements types.SpotifyProvider against the Spotify Web API.
type Handler struct {
	client *http.Client
}

// NewHandler creates a Handler that authenticates via OAuth2 refresh-token flow.
// Tokens are refreshed transparently by the oauth2 package.
func NewHandler(clientID, clientSecret, refreshToken string) (*Handler, error) {
	if clientID == "" || clientSecret == "" || refreshToken == "" {
		return nil, fmt.Errorf("clientID, clientSecret, and refreshToken are all required")
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			TokenURL: "https://accounts.spotify.com/api/token",
		},
	}
	tok := &oauth2.Token{
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(-time.Hour), // force a refresh on first call
	}

	return &Handler{client: cfg.Client(context.Background(), tok)}, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spotify API error %d: %s", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (h *Handler) put(ctx context.Context, path string, body any) error {
	return h.doJSON(ctx, http.MethodPut, path, body)
}

func (h *Handler) post(ctx context.Context, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, strings.NewReader(string(b)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spotify API error %d: %s", resp.StatusCode, string(raw))
	}
	if out != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (h *Handler) doJSON(ctx context.Context, method, path string, body any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = strings.NewReader(string(b))
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("spotify API error %d: %s", resp.StatusCode, string(raw))
}

// ── Search ────────────────────────────────────────────────────────────────────

type searchResponse struct {
	Tracks    struct{ Items []trackObject }    `json:"tracks"`
	Artists   struct{ Items []artistObject }   `json:"artists"`
	Albums    struct{ Items []albumObject }    `json:"albums"`
	Playlists struct{ Items []playlistObject } `json:"playlists"`
}

func (h *Handler) Search(ctx context.Context, query string, searchTypes []string, limit int) (*types.SearchResult, error) {
	if len(searchTypes) == 0 {
		searchTypes = []string{"track", "album", "artist", "playlist"}
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("type", strings.Join(searchTypes, ","))
	params.Set("limit", fmt.Sprintf("%d", limit))

	var resp searchResponse
	if err := h.get(ctx, "/search?"+params.Encode(), &resp); err != nil {
		return nil, err
	}

	result := &types.SearchResult{}
	for _, t := range resp.Tracks.Items {
		result.Tracks = append(result.Tracks, mapTrack(t))
	}
	for _, a := range resp.Artists.Items {
		result.Artists = append(result.Artists, mapArtist(a))
	}
	for _, a := range resp.Albums.Items {
		result.Albums = append(result.Albums, mapAlbum(a))
	}
	for _, p := range resp.Playlists.Items {
		result.Playlists = append(result.Playlists, mapPlaylist(p))
	}
	return result, nil
}

// ── Playback state ────────────────────────────────────────────────────────────

type playerState struct {
	IsPlaying            bool        `json:"is_playing"`
	Item                 trackObject `json:"item"`
	CurrentlyPlayingType string      `json:"currently_playing_type"`
	Device               struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Type          string `json:"type"`
		IsActive      bool   `json:"is_active"`
		IsRestricted  bool   `json:"is_restricted"`
		VolumePercent int    `json:"volume_percent"`
	} `json:"device"`
	ShuffleState bool   `json:"shuffle_state"`
	RepeatState  string `json:"repeat_state"`
	ProgressMs   int32  `json:"progress_ms"`
}

func (h *Handler) NowPlaying(ctx context.Context) (*types.PlaybackState, error) {
	var state playerState
	if err := h.get(ctx, "/me/player", &state); err != nil {
		// 204 = nothing playing
		if strings.Contains(err.Error(), "204") || strings.Contains(err.Error(), "no content") {
			return &types.PlaybackState{IsPlaying: false}, nil
		}
		return nil, err
	}

	ps := &types.PlaybackState{
		IsPlaying:   state.IsPlaying,
		Volume:      state.Device.VolumePercent,
		Shuffle:     state.ShuffleState,
		RepeatState: state.RepeatState,
		Progress:    state.ProgressMs,
		Device: &types.Device{
			ID:           state.Device.ID,
			Name:         state.Device.Name,
			Type:         state.Device.Type,
			IsActive:     state.Device.IsActive,
			IsRestricted: state.Device.IsRestricted,
			Volume:       state.Device.VolumePercent,
		},
	}
	if state.CurrentlyPlayingType == "track" {
		ps.Track = mapTrack(state.Item)
	}
	return ps, nil
}

// ── Playback control ──────────────────────────────────────────────────────────

func (h *Handler) PlayMusic(ctx context.Context, uri string, deviceID string) error {
	path := "/me/player/play"
	if deviceID != "" {
		path += "?device_id=" + url.QueryEscape(deviceID)
	}

	body := map[string]any{}
	if uri != "" {
		if strings.Contains(uri, ":track:") {
			body["uris"] = []string{uri}
		} else {
			body["context_uri"] = uri
		}
	}
	return h.put(ctx, path, body)
}

func (h *Handler) PausePlayback(ctx context.Context, deviceID string) error {
	path := "/me/player/pause"
	if deviceID != "" {
		path += "?device_id=" + url.QueryEscape(deviceID)
	}
	return h.doJSON(ctx, http.MethodPut, path, nil)
}

func (h *Handler) ResumePlayback(ctx context.Context, deviceID string) error {
	return h.PlayMusic(ctx, "", deviceID)
}

func (h *Handler) SkipToNext(ctx context.Context, deviceID string) error {
	path := "/me/player/next"
	if deviceID != "" {
		path += "?device_id=" + url.QueryEscape(deviceID)
	}
	return h.doJSON(ctx, http.MethodPost, path, nil)
}

func (h *Handler) SkipToPrevious(ctx context.Context, deviceID string) error {
	path := "/me/player/previous"
	if deviceID != "" {
		path += "?device_id=" + url.QueryEscape(deviceID)
	}
	return h.doJSON(ctx, http.MethodPost, path, nil)
}

func (h *Handler) SetVolume(ctx context.Context, percent int, deviceID string) error {
	path := fmt.Sprintf("/me/player/volume?volume_percent=%d", percent)
	if deviceID != "" {
		path += "&device_id=" + url.QueryEscape(deviceID)
	}
	return h.doJSON(ctx, http.MethodPut, path, nil)
}

func (h *Handler) TransferPlayback(ctx context.Context, deviceID string, play bool) error {
	body := map[string]any{
		"device_ids": []string{deviceID},
		"play":       play,
	}
	return h.put(ctx, "/me/player", body)
}

// ── Devices ───────────────────────────────────────────────────────────────────

type devicesResponse struct {
	Devices []struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Type          string `json:"type"`
		IsActive      bool   `json:"is_active"`
		IsRestricted  bool   `json:"is_restricted"`
		VolumePercent int    `json:"volume_percent"`
	} `json:"devices"`
}

func (h *Handler) ListDevices(ctx context.Context) ([]*types.Device, error) {
	var resp devicesResponse
	if err := h.get(ctx, "/me/player/devices", &resp); err != nil {
		return nil, err
	}
	devices := make([]*types.Device, 0, len(resp.Devices))
	for _, d := range resp.Devices {
		devices = append(devices, &types.Device{
			ID:           d.ID,
			Name:         d.Name,
			Type:         d.Type,
			IsActive:     d.IsActive,
			IsRestricted: d.IsRestricted,
			Volume:       d.VolumePercent,
		})
	}
	return devices, nil
}

// ── Playlists ─────────────────────────────────────────────────────────────────

type playlistsResponse struct {
	Items []playlistObject `json:"items"`
}

func (h *Handler) ListPlaylists(ctx context.Context, limit int) ([]*types.Playlist, error) {
	var resp playlistsResponse
	if err := h.get(ctx, fmt.Sprintf("/me/playlists?limit=%d", limit), &resp); err != nil {
		return nil, err
	}
	playlists := make([]*types.Playlist, 0, len(resp.Items))
	for _, p := range resp.Items {
		playlists = append(playlists, mapPlaylist(p))
	}
	return playlists, nil
}

func (h *Handler) GetPlaylist(ctx context.Context, playlistID string) (*types.Playlist, error) {
	var p playlistObject
	if err := h.get(ctx, "/playlists/"+playlistID, &p); err != nil {
		return nil, err
	}
	return mapPlaylist(p), nil
}

type playlistTracksResponse struct {
	Items []struct {
		Track trackObject `json:"track"`
	} `json:"items"`
}

func (h *Handler) GetPlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]*types.Track, error) {
	var resp playlistTracksResponse
	path := fmt.Sprintf("/playlists/%s/tracks?limit=%d&offset=%d", playlistID, limit, offset)
	if err := h.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	tracks := make([]*types.Track, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item.Track.ID != "" {
			tracks = append(tracks, mapTrack(item.Track))
		}
	}
	return tracks, nil
}

type meResponse struct {
	ID string `json:"id"`
}

func (h *Handler) CreatePlaylist(ctx context.Context, title, description string, public bool) (*types.Playlist, error) {
	var me meResponse
	if err := h.get(ctx, "/me", &me); err != nil {
		return nil, fmt.Errorf("failed to fetch current user: %w", err)
	}

	var created playlistObject
	body := map[string]any{
		"name":        title,
		"description": description,
		"public":      public,
	}
	if err := h.post(ctx, "/users/"+me.ID+"/playlists", body, &created); err != nil {
		return nil, err
	}
	return mapPlaylist(created), nil
}

func (h *Handler) AppendPlaylistTracks(ctx context.Context, playlistID string, uris []string) error {
	body := map[string]any{"uris": uris}
	var out any
	return h.post(ctx, "/playlists/"+playlistID+"/tracks", body, &out)
}

// ── Library ───────────────────────────────────────────────────────────────────

type recentlyPlayedResponse struct {
	Items []struct {
		Track trackObject `json:"track"`
	} `json:"items"`
}

func (h *Handler) ListRecentlyPlayed(ctx context.Context, limit int) ([]*types.Track, error) {
	var resp recentlyPlayedResponse
	if err := h.get(ctx, fmt.Sprintf("/me/player/recently-played?limit=%d", limit), &resp); err != nil {
		return nil, err
	}
	tracks := make([]*types.Track, 0, len(resp.Items))
	for _, item := range resp.Items {
		tracks = append(tracks, mapTrack(item.Track))
	}
	return tracks, nil
}

type savedTracksResponse struct {
	Items []struct {
		Track trackObject `json:"track"`
	} `json:"items"`
}

func (h *Handler) ListSavedTracks(ctx context.Context, limit, offset int) ([]*types.Track, error) {
	var resp savedTracksResponse
	path := fmt.Sprintf("/me/tracks?limit=%d&offset=%d", limit, offset)
	if err := h.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	tracks := make([]*types.Track, 0, len(resp.Items))
	for _, item := range resp.Items {
		tracks = append(tracks, mapTrack(item.Track))
	}
	return tracks, nil
}
