package types

import "context"

// SpotifyProvider defines the full interface for all Spotify operations.
// It is implemented by the spotify.Handler and consumed by the Service.
type SpotifyProvider interface {
	// Search queries Spotify for tracks, albums, artists, and playlists.
	Search(ctx context.Context, query string, searchTypes []string, limit int) (*SearchResult, error)

	// NowPlaying returns the current playback state.
	NowPlaying(ctx context.Context) (*PlaybackState, error)

	// PlayMusic starts playback of a URI on the given device (empty = active device).
	// Accepts track, album, artist, or playlist URIs.
	PlayMusic(ctx context.Context, uri string, deviceID string) error

	// PausePlayback pauses the current playback session.
	PausePlayback(ctx context.Context, deviceID string) error

	// ResumePlayback resumes a paused playback session.
	ResumePlayback(ctx context.Context, deviceID string) error

	// SkipToNext skips to the next track in the queue.
	SkipToNext(ctx context.Context, deviceID string) error

	// SkipToPrevious goes back to the previous track.
	SkipToPrevious(ctx context.Context, deviceID string) error

	// SetVolume sets the playback volume (0-100).
	SetVolume(ctx context.Context, percent int, deviceID string) error

	// TransferPlayback moves the active session to the given device.
	TransferPlayback(ctx context.Context, deviceID string, play bool) error

	// ListDevices returns all available Spotify Connect devices.
	ListDevices(ctx context.Context) ([]*Device, error)

	// ListPlaylists returns the current user's playlists (paginated).
	ListPlaylists(ctx context.Context, limit int) ([]*Playlist, error)

	// GetPlaylist fetches a single playlist by ID.
	GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error)

	// GetPlaylistTracks returns the tracks inside a playlist.
	GetPlaylistTracks(ctx context.Context, playlistID string, limit, offset int) ([]*Track, error)

	// CreatePlaylist creates a new playlist for the current user.
	CreatePlaylist(ctx context.Context, title, description string, public bool) (*Playlist, error)

	// AppendPlaylistTracks adds tracks to an existing playlist.
	AppendPlaylistTracks(ctx context.Context, playlistID string, uris []string) error

	// ListRecentlyPlayed returns the user's recently played tracks.
	ListRecentlyPlayed(ctx context.Context, limit int) ([]*Track, error)

	// ListSavedTracks returns tracks saved in the user's library.
	ListSavedTracks(ctx context.Context, limit, offset int) ([]*Track, error)
}
