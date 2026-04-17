package common

import (
	"log/slog"
	"os"
	"strings"
)

func NewLogger(level string) (*slog.Logger, error) {
	var slogLevel slog.Level
	if err := slogLevel.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		return nil, err
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}
