package internal

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Kavin-bm/Spotify-MCP/config"
	"github.com/Kavin-bm/Spotify-MCP/types"
)

// buildMCPInstructions returns the session instructions injected into every
// MCP client on connect. Adjusts wording based on whether a default device
// is configured.
func buildMCPInstructions(cfg *config.Config) string {
	if cfg.DeviceName != "" {
		return fmt.Sprintf(`Spotify music control service. Provides search, playback control, playlist management, and library access via the Spotify Web API.

Target playback device is pre-configured (%q). You do not need to call list_devices or pass device_id — it is resolved automatically.

Simplified play flow:
  1. search      — find the track/album/artist/playlist; get its URI
  2. play_music  — start playback with the URI (no device_id needed)
  3. now_playing — verify playback is active

Voice intent routing:
  "Play [artist/song/album]"     → search → play_music
  "What's playing"               → now_playing
  "Pause / Resume"               → pause_playback / resume_playback
  "Skip / Previous"              → skip_next / skip_previous
  "Set volume to X"              → set_volume
  "Play something similar"       → now_playing → search (by artist/genre) → play_music
  "Play my [name] playlist"      → list_playlists → play_music (playlist URI)
  "Play my liked songs"          → list_saved_tracks → play_music
  "Play my recent tracks"        → list_recently_played → play_music
  "Create a playlist from [X]"   → [fetch tracks] → create_playlist → append_playlist_tracks → play_music`, cfg.DeviceName)
	}

	return `Spotify music control service. Provides search, playback control, playlist management, and library access via the Spotify Web API.

No default device is configured. Use the full device-aware play flow:
  1. list_devices        — find the target device and record its ID
  2. transfer_playback   — activate the device silently (play=false) to avoid "no active device" errors
  3. set_volume          — set the volume level on the device
  4. play_music          — start playback with device_id and URI
  5. now_playing         — verify playback is active

Always pass device_id to all playback operations.

Voice intent routing:
  "Play [artist/song/album]"     → search → play flow
  "What's playing"               → now_playing
  "Pause / Resume"               → pause_playback / resume_playback
  "Skip / Previous"              → skip_next / skip_previous
  "Set volume to X"              → set_volume
  "Play something similar"       → now_playing → search (by artist/genre) → play flow
  "Play my [name] playlist"      → list_playlists → play flow using playlist URI
  "Play my liked songs"          → list_saved_tracks → play flow
  "Play my recent tracks"        → list_recently_played → play flow
  "Create a playlist from [X]"   → [fetch tracks] → create_playlist → append_playlist_tracks → play flow`
}

// ── Tool registration ─────────────────────────────────────────────────────────

func registerTools(s *mcp.Server, svc *Service) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search",
		Description: "Search Spotify for tracks, albums, artists, or playlists. Returns URIs you can pass to play_music.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[searchInput]) (*mcp.CallToolResultFor[searchOutput], error) {
		result, err := svc.Search(ctx, params.Arguments.Query, params.Arguments.Types, params.Arguments.Limit)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[searchOutput]{Content: []mcp.Content{
			&mcp.TextContent{Text: formatSearchResult(result)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "now_playing",
		Description: "Get the current Spotify playback state including track, artist, album, progress, and device.",
	}, func(ctx context.Context, _ *mcp.ServerSession, _ *mcp.CallToolParamsFor[struct{}]) (*mcp.CallToolResultFor[struct{}], error) {
		state, err := svc.NowPlaying(ctx)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: formatPlaybackState(state)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "play_music",
		Description: "Start playback of a Spotify URI (track, album, artist, or playlist). Optionally specify a device_id.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[playMusicInput]) (*mcp.CallToolResultFor[struct{}], error) {
		if err := svc.PlayMusic(ctx, params.Arguments.URI, params.Arguments.DeviceID); err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: "Playback started."},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "pause_playback",
		Description: "Pause the current Spotify playback session.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[deviceInput]) (*mcp.CallToolResultFor[struct{}], error) {
		if err := svc.PausePlayback(ctx, params.Arguments.DeviceID); err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: "Playback paused."},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "resume_playback",
		Description: "Resume a paused Spotify playback session.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[deviceInput]) (*mcp.CallToolResultFor[struct{}], error) {
		if err := svc.ResumePlayback(ctx, params.Arguments.DeviceID); err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: "Playback resumed."},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "skip_next",
		Description: "Skip to the next track in the Spotify queue.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[deviceInput]) (*mcp.CallToolResultFor[struct{}], error) {
		if err := svc.SkipNext(ctx, params.Arguments.DeviceID); err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: "Skipped to next track."},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "skip_previous",
		Description: "Go back to the previous track.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[deviceInput]) (*mcp.CallToolResultFor[struct{}], error) {
		if err := svc.SkipPrevious(ctx, params.Arguments.DeviceID); err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: "Skipped to previous track."},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_volume",
		Description: "Set the playback volume (0–100).",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[setVolumeInput]) (*mcp.CallToolResultFor[struct{}], error) {
		if err := svc.SetVolume(ctx, params.Arguments.VolumePercent, params.Arguments.DeviceID); err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Volume set to %d%%.", params.Arguments.VolumePercent)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "transfer_playback",
		Description: "Transfer the active Spotify session to a different device.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[transferInput]) (*mcp.CallToolResultFor[struct{}], error) {
		if err := svc.TransferPlayback(ctx, params.Arguments.DeviceID, params.Arguments.Play); err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: "Playback transferred."},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_devices",
		Description: "List all available Spotify Connect devices and their IDs.",
	}, func(ctx context.Context, _ *mcp.ServerSession, _ *mcp.CallToolParamsFor[struct{}]) (*mcp.CallToolResultFor[struct{}], error) {
		devices, err := svc.ListDevices(ctx)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: formatDevices(devices)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_playlists",
		Description: "List the current user's Spotify playlists (up to 50).",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[limitInput]) (*mcp.CallToolResultFor[struct{}], error) {
		playlists, err := svc.ListPlaylists(ctx, params.Arguments.Limit)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: formatPlaylists(playlists)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_playlist",
		Description: "Fetch details of a single playlist by ID.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[playlistIDInput]) (*mcp.CallToolResultFor[struct{}], error) {
		playlist, err := svc.GetPlaylist(ctx, params.Arguments.PlaylistID)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: formatPlaylist(playlist)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_playlist_tracks",
		Description: "Fetch the tracks in a playlist by ID.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[playlistTracksInput]) (*mcp.CallToolResultFor[struct{}], error) {
		tracks, err := svc.GetPlaylistTracks(ctx, params.Arguments.PlaylistID, params.Arguments.Limit, params.Arguments.Offset)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: formatTracks(tracks)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "create_playlist",
		Description: "Create a new Spotify playlist for the current user. Optionally seed it with track URIs.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[createPlaylistInput]) (*mcp.CallToolResultFor[struct{}], error) {
		playlist, err := svc.CreatePlaylist(ctx,
			params.Arguments.Title,
			params.Arguments.Description,
			params.Arguments.Public,
			params.Arguments.TrackURIs,
		)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Created playlist %q (URI: %s)", playlist.Name, playlist.URI)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "append_playlist_tracks",
		Description: "Add tracks to an existing playlist.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[appendTracksInput]) (*mcp.CallToolResultFor[struct{}], error) {
		if err := svc.AppendPlaylistTracks(ctx, params.Arguments.PlaylistID, params.Arguments.URIs); err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Added %d tracks to playlist.", len(params.Arguments.URIs))},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_recently_played",
		Description: "List the user's recently played tracks.",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[limitInput]) (*mcp.CallToolResultFor[struct{}], error) {
		tracks, err := svc.ListRecentlyPlayed(ctx, params.Arguments.Limit)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: formatTracks(tracks)},
		}}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_saved_tracks",
		Description: "List tracks saved in the user's Spotify library (liked songs).",
	}, func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[savedTracksInput]) (*mcp.CallToolResultFor[struct{}], error) {
		tracks, err := svc.ListSavedTracks(ctx, params.Arguments.Limit, params.Arguments.Offset)
		if err != nil {
			return nil, err
		}
		return &mcp.CallToolResultFor[struct{}]{Content: []mcp.Content{
			&mcp.TextContent{Text: formatTracks(tracks)},
		}}, nil
	})
}

// ── Input/output shapes ───────────────────────────────────────────────────────

type searchInput struct {
	Query string   `json:"query" jsonschema:"description=Search query string"`
	Types []string `json:"types,omitempty" jsonschema:"description=Content types to search: track album artist playlist"`
	Limit int      `json:"limit,omitempty" jsonschema:"description=Max results per type (default 10)"`
}

type searchOutput struct{}

type playMusicInput struct {
	URI      string `json:"uri" jsonschema:"description=Spotify URI to play (track album artist or playlist)"`
	DeviceID string `json:"device_id,omitempty" jsonschema:"description=Target device ID (optional if SPOTIFY_DEVICE_NAME is configured)"`
}

type deviceInput struct {
	DeviceID string `json:"device_id,omitempty" jsonschema:"description=Target device ID (optional if SPOTIFY_DEVICE_NAME is configured)"`
}

type setVolumeInput struct {
	VolumePercent int    `json:"volume_percent" jsonschema:"description=Volume level 0-100"`
	DeviceID      string `json:"device_id,omitempty" jsonschema:"description=Target device ID"`
}

type transferInput struct {
	DeviceID string `json:"device_id" jsonschema:"description=Target device ID to transfer playback to"`
	Play     bool   `json:"play,omitempty" jsonschema:"description=Whether to start playing immediately after transfer"`
}

type limitInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=Max items to return (default 20 max 50)"`
}

type playlistIDInput struct {
	PlaylistID string `json:"playlist_id" jsonschema:"description=Spotify playlist ID"`
}

type playlistTracksInput struct {
	PlaylistID string `json:"playlist_id" jsonschema:"description=Spotify playlist ID"`
	Limit      int    `json:"limit,omitempty" jsonschema:"description=Max tracks to return (default 50)"`
	Offset     int    `json:"offset,omitempty" jsonschema:"description=Pagination offset"`
}

type createPlaylistInput struct {
	Title       string   `json:"title" jsonschema:"description=Playlist title"`
	Description string   `json:"description,omitempty" jsonschema:"description=Playlist description"`
	Public      bool     `json:"public,omitempty" jsonschema:"description=Whether the playlist is public"`
	TrackURIs   []string `json:"track_uris,omitempty" jsonschema:"description=Optional initial track URIs to add"`
}

type appendTracksInput struct {
	PlaylistID string   `json:"playlist_id" jsonschema:"description=Spotify playlist ID"`
	URIs       []string `json:"uris" jsonschema:"description=Track URIs to append"`
}

type savedTracksInput struct {
	Limit  int `json:"limit,omitempty" jsonschema:"description=Max tracks to return (default 20)"`
	Offset int `json:"offset,omitempty" jsonschema:"description=Pagination offset"`
}

// ── Formatters ────────────────────────────────────────────────────────────────

func formatTracks(tracks []*types.Track) string {
	if len(tracks) == 0 {
		return "No tracks found."
	}
	out := fmt.Sprintf("%d track(s):\n", len(tracks))
	for i, t := range tracks {
		artists := ""
		for j, a := range t.Artists {
			if j > 0 {
				artists += ", "
			}
			artists += a.Name
		}
		out += fmt.Sprintf("  %d. %s — %s (URI: %s)\n", i+1, t.Name, artists, t.URI)
	}
	return out
}

func formatDevices(devices []*types.Device) string {
	if len(devices) == 0 {
		return "No devices found. Make sure a Spotify client is open."
	}
	out := fmt.Sprintf("%d device(s):\n", len(devices))
	for _, d := range devices {
		active := ""
		if d.IsActive {
			active = " [active]"
		}
		out += fmt.Sprintf("  - %s (%s)%s — ID: %s, Volume: %d%%\n", d.Name, d.Type, active, d.ID, d.Volume)
	}
	return out
}

func formatPlaylists(playlists []*types.Playlist) string {
	if len(playlists) == 0 {
		return "No playlists found."
	}
	out := fmt.Sprintf("%d playlist(s):\n", len(playlists))
	for i, p := range playlists {
		out += fmt.Sprintf("  %d. %s (%d tracks) — URI: %s\n", i+1, p.Name, p.TrackCount, p.URI)
	}
	return out
}

func formatPlaylist(p *types.Playlist) string {
	return fmt.Sprintf("Playlist: %s\nDescription: %s\nTracks: %d\nPublic: %v\nURI: %s\n",
		p.Name, p.Description, p.TrackCount, p.IsPublic, p.URI)
}

func formatPlaybackState(s *types.PlaybackState) string {
	if s == nil || !s.IsPlaying {
		return "Nothing is currently playing."
	}
	out := "Now Playing:\n"
	if s.Track != nil {
		artists := ""
		for i, a := range s.Track.Artists {
			if i > 0 {
				artists += ", "
			}
			artists += a.Name
		}
		out += fmt.Sprintf("  Track:  %s\n  Artist: %s\n  Album:  %s\n  URI:    %s\n",
			s.Track.Name, artists, s.Track.Album.Name, s.Track.URI)
	}
	if s.Device != nil {
		out += fmt.Sprintf("  Device: %s (%s)\n", s.Device.Name, s.Device.Type)
	}
	out += fmt.Sprintf("  Volume: %d%%\n  Shuffle: %v\n  Repeat: %s\n", s.Volume, s.Shuffle, s.RepeatState)
	return out
}

func formatSearchResult(r *types.SearchResult) string {
	if r == nil {
		return "No results."
	}
	out := ""
	if len(r.Tracks) > 0 {
		out += fmt.Sprintf("Tracks (%d):\n", len(r.Tracks))
		for i, t := range r.Tracks {
			artists := ""
			for j, a := range t.Artists {
				if j > 0 {
					artists += ", "
				}
				artists += a.Name
			}
			out += fmt.Sprintf("  %d. %s — %s (URI: %s)\n", i+1, t.Name, artists, t.URI)
		}
	}
	if len(r.Artists) > 0 {
		out += fmt.Sprintf("Artists (%d):\n", len(r.Artists))
		for i, a := range r.Artists {
			out += fmt.Sprintf("  %d. %s (URI: %s)\n", i+1, a.Name, a.URI)
		}
	}
	if len(r.Albums) > 0 {
		out += fmt.Sprintf("Albums (%d):\n", len(r.Albums))
		for i, a := range r.Albums {
			out += fmt.Sprintf("  %d. %s (URI: %s)\n", i+1, a.Name, a.URI)
		}
	}
	if len(r.Playlists) > 0 {
		out += fmt.Sprintf("Playlists (%d):\n", len(r.Playlists))
		for i, p := range r.Playlists {
			out += fmt.Sprintf("  %d. %s (%d tracks) (URI: %s)\n", i+1, p.Name, p.TrackCount, p.URI)
		}
	}
	if out == "" {
		return "No results found."
	}
	return out
}
