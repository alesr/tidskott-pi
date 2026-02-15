package raspberry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alesr/tidskott-core/pkg/interfaces"
)

var _ interfaces.CameraSource = (*RaspberryPiCamera)(nil)

type RaspberryPiCamera struct {
	config     interfaces.Config
	outputPath string
	logger     *slog.Logger

	cmd          *exec.Cmd
	running      bool
	mu           sync.RWMutex
	processState *os.ProcessState
}

func NewRaspberryPiCamera(logger *slog.Logger, cfg interfaces.Config, outPath string) (*RaspberryPiCamera, error) {
	return &RaspberryPiCamera{
		logger:     logger.With("component", "rpi_camera"),
		config:     cfg,
		outputPath: outPath,
	}, nil
}

func (c *RaspberryPiCamera) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	c.logger.Info(
		"Starting Raspberry Pi camera",
		"resolution", fmt.Sprintf("%dx%d", c.config.Width, c.config.Height),
		"fps", c.config.FPS,
		"bitrate", c.config.Bitrate,
		"output", c.outputPath,
	)

	c.cmd = exec.CommandContext(
		ctx,
		"rpicam-vid",
		"--width", fmt.Sprintf("%d", c.config.Width),
		"--height", fmt.Sprintf("%d", c.config.Height),
		"--framerate", fmt.Sprintf("%d", c.config.FPS),
		"--bitrate", fmt.Sprintf("%d", c.config.Bitrate),
		"--codec", c.config.Codec,
		"--profile", c.config.Profile,
		"--intra", fmt.Sprintf("%d", c.config.KeyframeInterval),
		"--quality", fmt.Sprintf("%d", c.config.Quality),
		"--output", c.outputPath,
		"--nopreview",
	)

	c.cmd.Stderr = os.Stderr

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start camera: %w", err)
	}

	c.logger.Info("Camera started", "pid", c.cmd.Process.Pid)
	c.running = true
	c.config.StartTime = time.Now()
	return nil
}

func (c *RaspberryPiCamera) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running || c.cmd == nil || c.cmd.Process == nil {
		c.running = false
		return nil
	}

	c.logger.Info("Stopping camera", "pid", c.cmd.Process.Pid)

	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		c.logger.Warn("Failed to send SIGTERM to camera process", "error", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- c.cmd.Wait()
	}()

	select {
	case err := <-done:
		c.processState = c.cmd.ProcessState
		c.running = false
		// TODO(alesr): use sentinel
		if err != nil && !strings.Contains(err.Error(), "killed") {
			c.logger.Error("Camera process exited with error", "error", err)
			return err
		}
		c.logger.Info("Camera stopped gracefully")
		return nil

	case <-func() <-chan time.Time {
		shutdownTimeout := 500 * time.Millisecond
		shutdownTimer := time.NewTimer(shutdownTimeout)
		defer shutdownTimer.Stop()
		return shutdownTimer.C
	}():
		//  didn't exit after grace period, use SIGKILL O_o
		c.logger.Warn("Camera not responding to SIGTERM, forcing termination")

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

func (c *RaspberryPiCamera) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *RaspberryPiCamera) GetConfig() interfaces.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

func (c *RaspberryPiCamera) GetOutputPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.outputPath
}

func (c *RaspberryPiCamera) GetName() string {
	return "Raspberry Pi Camera"
}

func NewRaspberryPiCameraFactory(logger *slog.Logger) interfaces.Factory {
	return func(outputPath string, config interfaces.Config) (interfaces.CameraSource, error) {
		return NewRaspberryPiCamera(logger, config, outputPath)
	}
}
