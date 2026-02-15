package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type (
	Config struct {
		Device DeviceConfig `toml:"device"`
		Camera CameraConfig `toml:"camera"`
		Buffer BufferConfig `toml:"buffer"`
		Upload UploadConfig `toml:"upload"`
		Auth   AuthConfig   `toml:"auth"`
	}

	DeviceConfig struct {
		ID   string `toml:"id"`
		Name string `toml:"name"`
	}

	CameraConfig struct {
		Width   int    `toml:"width"`
		Height  int    `toml:"height"`
		FPS     int    `toml:"fps"`
		Bitrate int    `toml:"bitrate"`
		Codec   string `toml:"codec"`
	}

	BufferConfig struct {
		WindowSeconds    int `toml:"window_seconds"`
		SnapshotDuration int `toml:"snapshot_duration"`
		SnapshotInterval int `toml:"snapshot_interval"`
	}

	UploadConfig struct {
		Endpoint          string `toml:"endpoint"`
		MaxRetries        int    `toml:"max_retries"`
		MaxConcurrent     int    `toml:"max_concurrent"`
		DeleteAfterUpload bool   `toml:"delete_after_upload"`
	}

	AuthConfig struct {
		Enabled      bool   `toml:"enabled"`
		Endpoint     string `toml:"endpoint"`
		ClientID     string `toml:"client_id"`
		ClientSecret string `toml:"client_secret"`
	}
)

func DefaultConfig() *Config {
	return &Config{
		Device: DeviceConfig{
			ID:   "tidskott-pi-device",
			Name: "tidskott Pi Camera",
		},
		Camera: CameraConfig{
			Width:   1920,
			Height:  1080,
			FPS:     30,
			Bitrate: 25000000,
			Codec:   "libx265",
		},
		Buffer: BufferConfig{
			WindowSeconds:    30,
			SnapshotDuration: 5,
			SnapshotInterval: 5,
		},
		Upload: UploadConfig{
			Endpoint:          "http://localhost:8080/upload",
			MaxRetries:        3,
			MaxConcurrent:     2,
			DeleteAfterUpload: true,
		},
		Auth: AuthConfig{
			Enabled:      false,
			Endpoint:     "/auth/token",
			ClientID:     "tidskott-client",
			ClientSecret: "tidskott-secret",
		},
	}
}

func DefaultConfigPath() string { return "config.toml" }

func LoadConfig(path string) (*Config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := toml.Unmarshal(contents, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return config, nil
}

// TODO(alesr): enum errors and split
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Device.ID) == "" {
		return errors.New("device.id cannot be empty")
	}
	if strings.TrimSpace(c.Device.Name) == "" {
		return errors.New("device.name cannot be empty")
	}

	if c.Camera.Width <= 0 {
		return errors.New("camera.width must be positive")
	}
	if c.Camera.Height <= 0 {
		return errors.New("camera.height must be positive")
	}
	if c.Camera.FPS <= 0 || c.Camera.FPS > 120 {
		return errors.New("camera.fps must be between 1 and 120")
	}
	if c.Camera.Bitrate < 0 {
		return errors.New("camera.bitrate cannot be negative")
	}
	if strings.TrimSpace(c.Camera.Codec) == "" {
		return errors.New("camera.codec cannot be empty")
	}

	if c.Buffer.WindowSeconds < 5 || c.Buffer.WindowSeconds > 60 {
		return errors.New("buffer.window_seconds must be between 5 and 60")
	}
	if c.Buffer.SnapshotDuration <= 0 {
		return errors.New("buffer.snapshot_duration must be positive")
	}
	if c.Buffer.SnapshotDuration > c.Buffer.WindowSeconds {
		return errors.New("buffer.snapshot_duration cannot exceed buffer.window_seconds")
	}
	if c.Buffer.SnapshotInterval <= 0 {
		return errors.New("buffer.snapshot_interval must be positive")
	}

	if strings.TrimSpace(c.Upload.Endpoint) == "" {
		return errors.New("upload.endpoint cannot be empty")
	}
	if !strings.HasPrefix(c.Upload.Endpoint, "http://") && !strings.HasPrefix(c.Upload.Endpoint, "https://") {
		return errors.New("upload.endpoint must start with http:// or https://")
	}
	if c.Upload.MaxRetries < 0 {
		return errors.New("upload.max_retries cannot be negative")
	}
	if c.Upload.MaxConcurrent <= 0 {
		return errors.New("upload.max_concurrent must be positive")
	}

	if c.Auth.Enabled {
		if strings.TrimSpace(c.Auth.Endpoint) == "" {
			return errors.New("auth.endpoint cannot be empty when auth is enabled")
		}
		if strings.TrimSpace(c.Auth.ClientID) == "" {
			return errors.New("auth.client_id cannot be empty when auth is enabled")
		}
		if strings.TrimSpace(c.Auth.ClientSecret) == "" {
			return errors.New("auth.client_secret cannot be empty when auth is enabled")
		}
	}
	return nil
}
