package logger

import (
	"io"
	"log/slog"
)

// New creates a new logger to use.
func New(writer io.Writer) *slog.Logger {
	return slog.New(NewHandler(writer, nil))
}
