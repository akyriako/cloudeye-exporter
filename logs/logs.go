package logs

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func InitLogger(debug bool) {
	levelInfo := slog.LevelInfo
	if debug {
		levelInfo = slog.LevelDebug
	}

	Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelInfo,
	}))

	slog.SetDefault(Logger)
}
