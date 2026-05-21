package types

// ArtistRef is a lightweight artist reference embedded inside a Track.
type ArtistRef struct {
	ID    string // Spotify artist ID
	Name  string // Human-readable artist name
	URI   string // spotify:artist:{id}
}

// AlbumRef is a lightweight album reference embedded inside a Track.
type AlbumRef struct {
	ID       string
	Name     string
	URI      string // spotify:album:{id}
	ImageURL string
}

// Track represents a Spotify music track.
type Track struct {
	ID       string
	Name     string     // Human-readable track title
	URI      string     // spotify:track:{id} — pass to PlayMusic
	ImageURL string
	Duration int32      // duration in milliseconds
	Artists  []ArtistRef
	Album    AlbumRef
}

// Artist represents a Spotify artist.
type Artist struct {
	ID   string
	Name string // Human-readable artist name
	URI  string // spotify:artist:{id} — pass to PlayMusic
}

// Album represents a Spotify album.
type Album struct {
	ID       string
	Name     string // Human-readable album title
	URI      string // spotify:album:{id} — pass to PlayMusic
	ImageURL string
}

// Playlist represents a Spotify playlist.
type Playlist struct {
	ID          string
	Name        string // Human-readable playlist title
	Description string
	IsPublic    bool
	TrackCount  int
	URI         string // spotify:playlist:{id} — pass to PlayMusic
	ImageURL    string
	OwnerID     string
	OwnerName   string
}

// Device represents a Spotify Connect playback device.
type Device struct {
	ID           string
	Name         string // Human-readable device name
	Type         string // "Computer", "Smartphone", "Speaker", etc.
	IsActive     bool
	IsRestricted bool
	Volume       int // 0–100
}

// PlaybackState represents the current state of the Spotify player.
type PlaybackState struct {
	IsPlaying   bool
	Track       *Track
	Device      *Device
	Volume      int    // 0–100
	Shuffle     bool
	RepeatState string // "off", "track", "context"
	Progress    int32  // progress in milliseconds
}

// SearchResult groups results across all content types.
type SearchResult struct {
	Tracks    []*Track
	Artists   []*Artist
	Albums    []*Album
	Playlists []*Playlist
}
