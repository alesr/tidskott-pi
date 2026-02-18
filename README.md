# Tidskott Pi

Raspberry Pi client for the Tidskott video streaming system.

## Overview

Tidskott Pi is a minimal Raspberry Pi client that captures video from the Raspberry Pi camera, maintains a rolling buffer of the most recent video frames, and uploads snapshots to a remote server.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://github.com/user-attachments/assets/d1d04c98-95d4-48fb-93cb-2e82440f715d">
  <source media="(prefers-color-scheme: light)" srcset="https://github.com/user-attachments/assets/d94bd4eb-6587-420f-bd45-53ee358afdb9">
  <img alt="Big Picture" src="https://github.com/user-attachments/assets/d94bd4eb-6587-420f-bd45-53ee358afdb9">
</picture>


> [!WARNING]
> ### 🚧 Work In Progress
> - [ ] **Wire dependencies**
> - [ ] **Tests**
> - [ ] **CI Pipeline**
> - [ ] **Secure HTTP Snapshots** (Triggering mechanism)
> - [ ] **Observability** (Grafana/Prometheus integration)

## Usage

### Building

```bash
# build for raspberry pi
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

The Raspberry PI client consists of:

1. **camera source**: built-in Raspberry Pi camera support using `rpicam-vid`
2. **video buffer**: maintains a rolling window of video frames using `tidskott-core`
3. **snapshot generator**: extracts video segments from the buffer on demand
4. **uploader**: uploads snapshots to a remote server using `tidskott-uploader`

## Dependencies

- **hardware**:
  - raspberry pi with camera module (for raspberry pi)
  - built-in or usb camera (for macos)
- **software**:
  - `rpicam-vid` (raspberry pi camera utility, for raspberry pi only)
  - `ffmpeg` (for macos camera support)
  - `tidskott-core` (core video buffering library)
  - `tidskott-uploader` (snapshot uploader)

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
