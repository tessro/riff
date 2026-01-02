# riff

A unified CLI for controlling music playback across Spotify and Sonos devices.

## Features

- **Spotify Control**: Play, pause, skip, seek, volume, queue management
- **Sonos Control**: Device discovery, playback control, speaker grouping
- **Unified Interface**: Same commands work across both platforms
- **Interactive Wizards**: Fuzzy search for tracks, device picker
- **Tail Mode**: Watch playback changes in real-time
- **JSON Output**: Script-friendly output with `--json` flag

## Installation

### From Source

```bash
go install github.com/tessro/riff/cmd/riff@latest
```

### From Releases

Download the latest release from the [releases page](https://github.com/tessro/riff/releases).

## Quick Start

```bash
# Authenticate with Spotify
riff auth login

# Play a track
riff play "bohemian rhapsody"

# Check what's playing
riff status

# Control playback
riff pause
riff next
riff prev

# List devices
riff devices

# Follow playback in real-time
riff tail
```

## Commands

### Playback

```bash
riff play [query]       # Play a track, album, or playlist
riff play --album [q]   # Play an album
riff play --playlist [q] # Play a playlist
riff pause              # Pause playback
riff next               # Skip to next track
riff prev               # Go to previous track
riff seek [position]    # Seek to position (e.g., "1:30")
riff volume [0-100]     # Set volume
```

### Status & Queue

```bash
riff status             # Show current playback
riff queue              # Show playback queue
riff queue add [uri]    # Add track to queue
```

### Devices

```bash
riff devices            # List available devices
riff devices transfer   # Transfer playback to device
```

### Sonos Groups

```bash
riff group list         # List speaker groups
riff group add          # Add speaker to group
riff group remove       # Remove speaker from group
```

### Authentication

```bash
riff auth login         # Authenticate with Spotify
riff auth status        # Check auth status
riff auth logout        # Clear stored credentials
```

### Tail Mode

```bash
riff tail               # Follow playback changes
riff tail -t            # Show timestamps
riff tail --no-emoji    # Disable emoji output
```

## Configuration

Create `~/.riffrc` or `~/.config/riff/config.toml`:

```toml
[spotify]
client_id = "your-client-id"

[sonos]
default_room = "Living Room"

[defaults]
volume = 50
shuffle = false
```

See `.riffrc.example` for all options.

## Global Flags

```
-c, --config    Config file path
-j, --json      JSON output
-v, --verbose   Verbose output
```

## Shell Completion

```bash
# Bash
riff completion bash > /etc/bash_completion.d/riff

# Zsh
riff completion zsh > "${fpath[1]}/_riff"

# Fish
riff completion fish > ~/.config/fish/completions/riff.fish
```

## License

MIT
