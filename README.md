# Tidskott Pi

Raspberry Pi client for the Tidskott video streaming system.

## Overview

Tidskott Pi is a minimal Raspberry Pi client that captures video from the Raspberry Pi camera, maintains a rolling buffer of the most recent video frames, and uploads snapshots to a remote server.

## Features

- **Single Configuration File**: All configuration in one TOML file
- **Minimal Footprint**: Optimized for Raspberry Pi
- **Efficient**: Uses in-memory buffering to minimize disk I/O
- **Robust**: Comprehensive error handling and retry logic
- **Authentication**: Supports JWT-based authentication
- **Validation**: Built-in configuration validation

## Usage

### Building

```bash
# build for Raspberry Pi (ARMv7)
make build-pi

# build for local development
make build
```

### Running

```bash
# run with default configuration
./bin/tidskott-pi

# or, with custom config
./bin/tidskott-pi --config /path/to/config.toml
```

## Configuration

The client uses a single configuration file named `config.toml` in the current directory. Use `--config` to point to a different file.

### Configuration Options

| Section | Option | Description | Default |
|---------|--------|-------------|---------|
| device | id | Device identifier | "tidskott-pi-device" |
| device | name | Human-readable device name | "tidskott Pi Camera" |
| camera | width | Frame width in pixels | 1920 |
| camera | height | Frame height in pixels | 1080 |
| camera | fps | Frames per second | 30 |
| camera | bitrate | Target bitrate in bits per second | 25000000 |
| camera | codec | Video codec | "libx265" |
| buffer | window_seconds | Rolling window size in seconds (5-60) | 30 |
| buffer | snapshot_duration | Snapshot duration in seconds | 5 |
| buffer | snapshot_interval | Interval between snapshots in seconds | 5 |
| upload | endpoint | Server endpoint for uploads | "http://localhost:8080/upload" |
| upload | max_retries | Maximum retry attempts for failed uploads | 3 |
| upload | max_concurrent | Maximum concurrent uploads | 2 |
| upload | delete_after_upload | Delete snapshots after successful upload | true |
| auth | enabled | Enable authentication | false |
| auth | endpoint | Authentication endpoint | "/auth/token" |
| auth | client_id | Client ID for authentication | "tidskott-client" |
| auth | client_secret | Client secret for authentication | "tidskott-secret" |

## Architecture

The Raspberry Pi client consists of:

1. **Camera Source**: Built-in Raspberry Pi camera support using `rpicam-vid`
2. **Video Buffer**: Maintains a rolling window of video frames using `tidskott-core`
3. **Snapshot Generator**: Extracts video segments from the buffer on demand
4. **Uploader**: Uploads snapshots to a remote server using `tidskott-uploader`

## Dependencies

- **Hardware**:
  - Raspberry Pi with camera module (for Raspberry Pi)
  - Built-in or USB camera (for macOS)
- **Software**:
  - `rpicam-vid` (Raspberry Pi camera utility, for Raspberry Pi only)
  - `ffmpeg` (for macOS camera support)
  - `tidskott-core` (Core video buffering library)
  - `tidskott-uploader` (Snapshot uploader)

## macOS Development

To use `tidskott-pi` on macOS for local development:

1. Install FFmpeg with `avfoundation` support:
   ```bash
   brew install ffmpeg
   ```

2. Grant camera permissions to your terminal app in:
   **System Preferences > Security & Privacy > Camera**.

3. Run `tidskott-pi`:
   ```bash
   ./bin/tidskott-pi
   ```

## TODO

```
- Trigger snapshot via HTTP (secure)
- Status API
- Tests
```
