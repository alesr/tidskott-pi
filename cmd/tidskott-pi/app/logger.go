package app

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func setupLogger() (*slog.Logger, *os.File, error) {
	logDir := "logs"

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	logFile, err := os.OpenFile(
		filepath.Join(
			logDir,
			fmt.Sprintf(
				"tidskott-pi_%s.log",
				time.Now().Format("20060102_150405"),
			),
		),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file: %w", err)
	}

	multiHandler := slog.NewTextHandler(
		io.MultiWriter(os.Stdout, logFile), &slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
	)
	return slog.New(multiHandler), logFile, nil
}
