package spotify

import (
	"fmt"

	"github.com/Kavin-bm/Spotify-MCP/types"
)

// ── raw API response shapes ───────────────────────────────────────────────────

type imageObject struct {
	URL string `json:"url"`
}

type artistObject struct {
	ID   string        `json:"id"`
	Name string        `json:"name"`
	URI  string        `json:"uri"`
	Images []imageObject `json:"images"`
}

type albumObject struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	URI     string        `json:"uri"`
	Images  []imageObject `json:"images"`
	Artists []artistObject `json:"artists"`
}

type trackObject struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	URI        string        `json:"uri"`
	DurationMs int32         `json:"duration_ms"`
	Album      albumObject   `json:"album"`
	Artists    []artistObject `json:"artists"`
}

type playlistObject struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URI         string `json:"uri"`
	Public      bool   `json:"public"`
	Tracks      struct {
		Total int `json:"total"`
	} `json:"tracks"`
	Images []imageObject `json:"images"`
	Owner  struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"owner"`
}

// ── mappers ───────────────────────────────────────────────────────────────────

func mapTrack(t trackObject) *types.Track {
	track := &types.Track{
		ID:       t.ID,
		Name:     t.Name,
		URI:      t.URI,
		Duration: t.DurationMs,
		Album: types.AlbumRef{
			ID:   t.Album.ID,
			Name: t.Album.Name,
			URI:  t.Album.URI,
		},
	}
	if len(t.Album.Images) > 0 {
		track.ImageURL = t.Album.Images[0].URL
		track.Album.ImageURL = t.Album.Images[0].URL
	}
	for _, a := range t.Artists {
		track.Artists = append(track.Artists, types.ArtistRef{
			ID:   a.ID,
			Name: fmt.Sprintf("artists/%s", a.ID),
			URI:  a.URI,
		})
	}
	return track
}

func mapArtist(a artistObject) *types.Artist {
	return &types.Artist{
		ID:   a.ID,
		Name: a.Name,
		URI:  a.URI,
	}
}

func mapAlbum(a albumObject) *types.Album {
	album := &types.Album{
		ID:   a.ID,
		Name: a.Name,
		URI:  a.URI,
	}
	if len(a.Images) > 0 {
		album.ImageURL = a.Images[0].URL
	}
	return album
}

func mapPlaylist(p playlistObject) *types.Playlist {
	pl := &types.Playlist{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		IsPublic:    p.Public,
		TrackCount:  p.Tracks.Total,
		URI:         p.URI,
		OwnerID:     p.Owner.ID,
		OwnerName:   p.Owner.DisplayName,
	}
	if len(p.Images) > 0 {
		pl.ImageURL = p.Images[0].URL
	}
	return pl
}
