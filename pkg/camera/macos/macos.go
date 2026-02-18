package macos

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alesr/tidskott-core/pkg/interfaces"
)

var _ interfaces.CameraSource = (*MacOSCamera)(nil)

type MacOSCamera struct {
	// config
	logger     *slog.Logger
	config     interfaces.Config
	outputPath string
	deviceID   string // format "INDEX:AUDIO" (e.g., "0:none")

	// runtime state
	cmd          *exec.Cmd
	running      bool
	mu           sync.RWMutex
	processState *os.ProcessState
}

func isNamedPipe(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeNamedPipe != 0
}

func NewMacOSCameraFactory(logger *slog.Logger, deviceID string) interfaces.Factory {
	return func(outputPath string, config interfaces.Config) (interfaces.CameraSource, error) {
		return NewMacOSCamera(logger, config, deviceID, outputPath)
	}
}

func NewMacOSCamera(logger *slog.Logger, cfg interfaces.Config, deviceID, outputPath string) (*MacOSCamera, error) {
	// ensure .ts extension for MPEG-TS format
	if !strings.HasSuffix(outputPath, ".ts") {
		outputPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".ts"
	}
	return &MacOSCamera{
		logger:     logger.With("component", "macos_camera"),
		config:     cfg,
		outputPath: outputPath,
		deviceID:   deviceID,
	}, nil
}

func (c *MacOSCamera) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	c.logger.Info(
		"Starting macOS camera",
		"device", c.deviceID,
		"resolution", fmt.Sprintf("%dx%d", c.config.Width, c.config.Height),
		"fps", c.config.FPS,
		"output", c.outputPath,
	)

	outputDir := filepath.Dir(c.outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// rm existing output file if it exists (skip if fifo)
	if _, err := os.Stat(c.outputPath); err == nil {
		if isNamedPipe(c.outputPath) {
			c.logger.Debug("Output path is a fifo, skipping removal", "path", c.outputPath)
		} else if err := os.Remove(c.outputPath); err != nil {
			c.logger.Warn("Failed to remove existing output file", "path", c.outputPath, "error", err)
		}
	}

	args := []string{
		"-hide_banner",
		"-f", "avfoundation",
		"-framerate", fmt.Sprintf("%d", c.config.FPS),
		"-pixel_format", "uyvy422", // native format for macOS camera
		"-video_size", fmt.Sprintf("%dx%d", c.config.Width, c.config.Height),
		"-thread_queue_size", "1024", // larger thread queue for stability
		"-i", c.deviceID,
		"-c:v", c.config.Codec,
		"-profile:v", c.config.Profile,
		"-preset", "ultrafast", // for live recording
		"-tune", "zerolatency", // for real-time
		"-b:v", fmt.Sprintf("%d", c.config.Bitrate),
		"-maxrate", fmt.Sprintf("%d", c.config.Bitrate),
		"-bufsize", fmt.Sprintf("%d", c.config.Bitrate*2),
		"-g", fmt.Sprintf("%d", c.config.KeyframeInterval),
		"-pix_fmt", "yuv420p", // ensure compatibility
		"-movflags", "+faststart", // pptimize for network streaming
		"-f", "mpegts", // use MPEG-TS format which is better for live streaming
		"-flush_packets", "1", // flush packets immediately
		"-muxdelay", "0", // no delay in muxing
		"-muxpreload", "0", // no preload in muxing
		"-y", // overwrite out file
		c.outputPath,
	}

	c.cmd = exec.CommandContext(ctx, "ffmpeg", args...)
	c.cmd.Env = append(os.Environ(), "AVFOUNDATION_SKIP_AUTHENTICATION=1")

	// stderr pipe for monitoring output

	stderr, err := c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	outputChan := make(chan string, 10)
	errorChan := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			outputChan <- line

			// log important messages
			// TODO(alesr): use sentinel
			if strings.Contains(line, "Error") || strings.Contains(line, "error") {
				c.logger.Error("Camera error", "message", line)
			} else if strings.Contains(line, "Warning") || strings.Contains(line, "warning") {
				c.logger.Warn("Camera warning", "message", line)
			}
		}
		if err := scanner.Err(); err != nil {
			errorChan <- err
		}
		close(outputChan)
	}()

	if err := c.cmd.Start(); err != nil {
		c.logger.Error(
			"Failed to start macOS camera",
			"error", err,
			"suggestion", "Check System Preferences > Security & Privacy > Camera permissions",
		)
		return fmt.Errorf("could not start camera: %w", err)
	}

	// wait for camera to start and create output file
	startTimeout := 5 * time.Second
	startTimer := time.NewTimer(startTimeout)
	fileCheckTicker := time.NewTicker(100 * time.Millisecond)

	defer fileCheckTicker.Stop()
	defer startTimer.Stop()

	// mnitor for startup
	var fileCreated, startupFailed bool

	for !fileCreated && !startupFailed {
		select {
		case <-ctx.Done():
			c.cmd.Process.Kill()
			return ctx.Err()

		case <-startTimer.C:
			c.logger.Warn("Timeout waiting for camera output file",
				"path", c.outputPath,
				"timeout", startTimeout.String())
			startupFailed = true

		case <-fileCheckTicker.C:
			// check if the file exists and has data
			if stat, err := os.Stat(c.outputPath); err == nil {
				if stat.Mode()&os.ModeNamedPipe != 0 {
					fileCreated = true
					c.logger.Info(
						"Camera output fifo ready",
						"path", c.outputPath,
					)
				} else if stat.Size() > 0 {
					fileCreated = true
					c.logger.Info(
						"Camera output file created",
						"path", c.outputPath,
						"size", stat.Size(),
					)
				}
			}

		case line, ok := <-outputChan:
			if !ok {
				continue
			}

			// check for error messages that indicate startup failure
			if strings.Contains(line, "Cannot open") ||
				strings.Contains(line, "Could not initialize") ||
				strings.Contains(line, "Permission denied") {
				c.logger.Error("Camera startup failed", "message", line)
				startupFailed = true
			}

			// check for success indicators
			if strings.Contains(line, "Stream mapping:") {
				c.logger.Info("Camera successfully initialized streams")
			}
		}
	}

	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		return fmt.Errorf("camera process exited immediately - check camera permissions")
	}

	if startupFailed && !fileCreated {
		c.cmd.Process.Kill()
		return fmt.Errorf("camera failed to create output file within timeout period")
	}

	c.logger.Info("Camera started", "pid", c.cmd.Process.Pid)
	c.running = true
	c.config.StartTime = time.Now()
	return nil
}

func (c *MacOSCamera) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running || c.cmd == nil || c.cmd.Process == nil {
		c.running = false
		return nil
	}

	c.logger.Info("Stopping camera", "pid", c.cmd.Process.Pid)

	// first try a graceful shutdown with SIGINT
	if err := c.cmd.Process.Signal(syscall.SIGINT); err != nil {
		c.logger.Warn("Failed to send SIGINT to camera process", "error", err)
		// fall back to SIGTERM
		if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			c.logger.Warn("Failed to send SIGTERM to camera process", "error", err)
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case err := <-done:
		c.processState = c.cmd.ProcessState
		c.running = false
		if err != nil && !strings.Contains(err.Error(), "killed") {
			return fmt.Errorf("camera process exited with error: %w", err)
		}
		return nil

	case <-func() <-chan time.Time {
		shutdownTimeout := 2 * time.Second
		shutdownTimer := time.NewTimer(shutdownTimeout)
		defer shutdownTimer.Stop()
		return shutdownTimer.C
	}():
		c.logger.Warn("Camera not responding to signals, forcing termination")

		if err := c.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill camera process: %w", err)
		}

		<-done
		c.processState = c.cmd.ProcessState
		c.running = false
		c.logger.Info("Camera forcefully terminated")
		return nil

	case <-ctx.Done():
		c.cmd.Process.Kill()
		c.running = false
		return ctx.Err()
	}
}

func (c *MacOSCamera) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *MacOSCamera) GetConfig() interfaces.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

func (c *MacOSCamera) GetOutputPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.outputPath
}

func (c *MacOSCamera) GetName() string { return "macOS Camera" }
