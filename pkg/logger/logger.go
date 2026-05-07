package logger

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

func Init(env string) {
	if env == "development" {
		slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelDebug,
		})))
	} else {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})))
	}
}
