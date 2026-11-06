package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

type Handler struct {
	config       Config
	groupOrAttrs []groupOrAttrs

	mutex  *sync.Mutex
	writer io.Writer
}

func NewHandler(writer io.Writer, config *Config) *Handler {
	result := Handler{
		mutex:  &sync.Mutex{},
		writer: writer,
	}
	if config != nil {
		result.config = *config
	}
	if result.config.Level == nil {
		result.config.Level = slog.LevelInfo
	}
	return &result
}

// Handler implements slog.Handler.
var _ slog.Handler = (*Handler)(nil)

func (h Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.config.Level.Level()
}

func (h Handler) Handle(ctx context.Context, record slog.Record) error {
	buffer := make([]byte, 0, 1024)
	buffer = record.Time.AppendFormat(buffer, "15:04:05")
	buffer = append(buffer, ' ')

	switch record.Level {
	case slog.LevelDebug:
		buffer = append(buffer, []byte("D ")...)
	case slog.LevelInfo:
		buffer = append(buffer, []byte("I ")...)
	case slog.LevelWarn:
		buffer = append(buffer, []byte("W ")...)
	case slog.LevelError:
		buffer = append(buffer, []byte("E ")...)
	default:
		buffer = append(buffer, []byte(record.Level.String())...)
		buffer = append(buffer, ' ')
	}

	buffer = append(buffer, []byte(record.Message)...)

	firstAttribute := true
	for _, goa := range h.groupOrAttrs {
		for _, attr := range goa.attrs {
			if firstAttribute {
				buffer = append(buffer, []byte(" (")...)
				firstAttribute = false
			} else {
				buffer = append(buffer, []byte(", ")...)
			}
			attr.Value = attr.Value.Resolve()
			buffer = fmt.Appendf(buffer, "%s: %q", attr.Key, attr.Value)
		}
	}
	record.Attrs(func(attr slog.Attr) bool {
		if firstAttribute {
			buffer = append(buffer, []byte(" (")...)
			firstAttribute = false
		} else {
			buffer = append(buffer, []byte(", ")...)
		}

		attr.Value = attr.Value.Resolve()
		buffer = fmt.Appendf(buffer, "%s: %q", attr.Key, attr.Value)
		return true
	})
	if !firstAttribute {
		buffer = append(buffer, ')')
	}
	buffer = append(buffer, '\n')

	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, err := h.writer.Write(buffer); err != nil {
		return err
	}
	return nil
}

func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func (h Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

func (h Handler) withGroupOrAttrs(goa groupOrAttrs) *Handler {
	result := h
	result.groupOrAttrs = make([]groupOrAttrs, len(h.groupOrAttrs)+1)
	copy(result.groupOrAttrs, h.groupOrAttrs)
	result.groupOrAttrs[len(result.groupOrAttrs)-1] = goa
	return &result
}

type groupOrAttrs struct {
	group string
	attrs []slog.Attr
}
