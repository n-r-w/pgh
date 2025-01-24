package pgh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// ILogger interface for logging.
type ILogger interface {
	Debugf(ctx context.Context, format string, args ...any)
	Infof(ctx context.Context, format string, args ...any)
	Warningf(ctx context.Context, format string, args ...any)
	Errorf(ctx context.Context, format string, args ...any)
}

// SlogLogger implements ILogger interface using slog.Logger.
type SlogLogger struct {
	l   *slog.Logger
	msg string
}

// NewSlogLogger returns new SlogLogger.
func NewSlogLogger(l *slog.Logger, msg string) (*SlogLogger, error) {
	if msg == "" {
		return nil, errors.New("NewSlogLogger: msg cannot be empty")
	}

	return &SlogLogger{
		l:   l,
		msg: msg,
	}, nil
}

func (s *SlogLogger) Debugf(ctx context.Context, format string, args ...any) {
	s.l.DebugContext(ctx, s.msg, slog.String("message", fmt.Sprintf(format, args...)))
}

func (s *SlogLogger) Infof(ctx context.Context, format string, args ...any) {
	s.l.InfoContext(ctx, s.msg, slog.String("message", fmt.Sprintf(format, args...)))
}

func (s *SlogLogger) Warningf(ctx context.Context, format string, args ...any) {
	s.l.WarnContext(ctx, s.msg, slog.String("message", fmt.Sprintf(format, args...)))
}

func (s *SlogLogger) Errorf(ctx context.Context, format string, args ...any) {
	s.l.ErrorContext(ctx, s.msg, slog.String("message", fmt.Sprintf(format, args...)))
}
