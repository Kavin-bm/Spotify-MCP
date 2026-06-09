# 🎵 Spotify MCP Server

A **Model Context Protocol (MCP)** server that exposes the Spotify Web API as a set of AI-callable tools. Connect any MCP-compatible AI assistant (Claude, Cursor, etc.) to control Spotify playback, manage playlists, search music, and more — all through natural language.

---

## ✨ Features

| Category | Tools |
|---|---|
| 🔍 **Search** | `search` — tracks, albums, artists, playlists |
| ▶️ **Playback** | `play_music`, `pause_playback`, `resume_playback`, `skip_next`, `skip_previous` |
| 🔊 **Device** | `list_devices`, `transfer_playback`, `set_volume` |
| 📋 **Playlists** | `list_playlists`, `get_playlist`, `get_playlist_tracks`, `create_playlist`, `append_playlist_tracks` |
| 📚 **Library** | `list_recently_played`, `list_saved_tracks` |
| 📡 **State** | `now_playing` |

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  AI Assistant (MCP Client)           │
│            (Claude / Cursor / any MCP client)        │
└───────────────────────┬─────────────────────────────┘
                        │ MCP over SSE (HTTP)
                        ▼
┌─────────────────────────────────────────────────────┐
│                  Spotify MCP Server                  │
│                                                      │
│  ┌──────────────┐  ┌────────────┐  ┌─────────────┐  │
│  │   MCP Tools  │  │  Service   │  │   Handler   │  │
│  │  (tools.go)  │─▶│(service.go)│─▶│(handler.go) │  │
│  └──────────────┘  └────────────┘  └──────┬──────┘  │
│                                           │         │
└───────────────────────────────────────────┼─────────┘
                                            │ HTTPS + OAuth2
                                            ▼
                              ┌─────────────────────────┐
                              │   Spotify Web API        │
                              └─────────────────────────┘
```

**Key design decisions:**
- **Stateless service layer** — device ID resolution is handled transparently with in-memory caching. If a `SPOTIFY_DEVICE_NAME` is configured, callers never need to pass `device_id`.
- **OAuth2 refresh-token flow** — tokens are rotated automatically. No cron jobs, no manual rotation.
- **Clean interfaces** — `types.SpotifyProvider` decouples the service from the Spotify SDK, making the handler fully testable with mocks.

---

## Quick Start

### 1. Get Spotify credentials

1. Go to [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) and create an app.
2. Add `http://localhost:8888/callback` as a redirect URI.
3. Note your **Client ID** and **Client Secret**.
4. Use the [Spotify OAuth guide](https://developer.spotify.com/documentation/web-api/concepts/authorization) to generate a **refresh token** with these scopes:

```
user-read-playback-state
user-modify-playback-state
user-read-currently-playing
playlist-read-private
playlist-modify-public
playlist-modify-private
user-library-read
user-read-recently-played
```

### 2. Configure environment

```bash
cp .env.example .env
# Fill in your credentials:
# SPOTIFY_CLIENT_ID=...
# SPOTIFY_CLIENT_SECRET=...
# SPOTIFY_REFRESH_TOKEN=...
```

### 3. Run the server

**Option A — Go directly:**
```bash
go run .
```

**Option B — Docker:**
```bash
docker-compose up
```

The MCP server starts on `http://localhost:8080`.

---

## MCP Client Configuration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "spotify": {
      "url": "http://localhost:8080/sse"
    }
  }
}
```

### Cursor / VS Code

Add to your MCP config:

```json
{
  "spotify": {
    "url": "http://localhost:8080/sse"
  }
}
```

---

## Tool Reference

### `search`
Search Spotify for any content type.

| Param | Type | Description |
|---|---|---|
| `query` | string | Search query |
| `types` | string[] | `track`, `album`, `artist`, `playlist` (default: all) |
| `limit` | int | Max results per type (default: 10) |

### `play_music`
Start playback of a Spotify URI.

| Param | Type | Description |
|---|---|---|
| `uri` | string | `spotify:track:…`, `spotify:album:…`, `spotify:playlist:…` |
| `device_id` | string | Target device (optional if `SPOTIFY_DEVICE_NAME` is set) |

### `now_playing`
Returns current playback state — track, artist, album, device, volume, progress.

### `set_volume`
| Param | Type | Description |
|---|---|---|
| `volume_percent` | int | 0–100 |
| `device_id` | string | Optional |

### `create_playlist`
| Param | Type | Description |
|---|---|---|
| `title` | string | Playlist name |
| `description` | string | Optional description |
| `public` | bool | Public or private |
| `track_uris` | string[] | Optional initial tracks |

> See all tools in [`internal/tools.go`](internal/tools.go).

---

## Device Auto-Resolution

Set `SPOTIFY_DEVICE_NAME` to the name of your Spotify Connect device (e.g. the name configured in `librespot --name`). The server resolves the device ID on first use and caches it in memory. If a playback call fails with a device-related error, the cache is automatically invalidated and the device is re-resolved.

Without `SPOTIFY_DEVICE_NAME`, you must call `list_devices` first and pass `device_id` to all playback calls.

---

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `SPOTIFY_CLIENT_ID` | ✅ | Spotify app client ID |
| `SPOTIFY_CLIENT_SECRET` | ✅ | Spotify app client secret |
| `SPOTIFY_REFRESH_TOKEN` | ✅ | OAuth2 refresh token |
| `SPOTIFY_DEVICE_NAME` | — | Auto-resolve device by name |
| `SPOTIFY_DEVICE_ID` | — | Explicit device ID (overrides name) |
| `PORT` | — | HTTP port (default: `8080`) |

---

## Project Structure

```
.
├── main.go                    # Entrypoint
├── config/
│   └── config.go              # Config loaded from env vars
├── types/
│   ├── interfaces.go          # SpotifyProvider interface
│   └── spotify.go             # Domain types (Track, Playlist, Device, …)
├── internal/
│   ├── server.go              # MCP HTTP server lifecycle
│   ├── service.go             # Business logic + device resolution
│   ├── tools.go               # MCP tool registration + formatters
│   └── spotify/
│       ├── handler.go         # Spotify Web API client (OAuth2)
│       └── mapper.go          # API response → domain type mappers
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

---

## License

MIT
