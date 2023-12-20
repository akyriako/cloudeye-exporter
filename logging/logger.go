package logging

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func InitLogger(debug bool) {
	levelInfo := slog.LevelInfo
	if debug {
		levelInfo = slog.LevelDebug
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelInfo,
	}))

	slog.SetDefault(logger)
}
