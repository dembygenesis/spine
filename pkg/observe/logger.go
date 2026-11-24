// Package observe is the logging seam: one interface, a production
// implementation, and a test double that records what was logged.
package observe

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

// Field is one key/value pair attached to a log line.
type Field struct {
	Key   string
	Value any
}

// F builds a Field.
func F(key string, value any) Field { return Field{Key: key, Value: value} }

// Logger is what the app logs through. Code depends on this, never on slog, so
// tests can swap in a TestLogger and assert on what was logged.
type Logger interface {
	Info(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, err error, fields ...Field)
}

// ── slog ──────────────────────────────────────────────────────────────────────

// SlogLogger writes JSON lines to stdout.
type SlogLogger struct{ l *slog.Logger }

// NewSlog returns the production logger.
func NewSlog() *SlogLogger {
	return &SlogLogger{l: slog.New(slog.NewJSONHandler(os.Stdout, nil))}
}

func (s *SlogLogger) Info(ctx context.Context, msg string, fields ...Field) {
	s.l.InfoContext(ctx, msg, attrs(fields)...)
}

func (s *SlogLogger) Error(ctx context.Context, msg string, err error, fields ...Field) {
	s.l.ErrorContext(ctx, msg, append(attrs(fields), "error", err)...)
}

func attrs(fields []Field) []any {
	out := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		out = append(out, f.Key, f.Value)
	}
	return out
}

// ── test double ───────────────────────────────────────────────────────────────

// LogEntry is one recorded error.
type LogEntry struct {
	Message string
	Error   error
	Fields  map[string]any
}

// TestLogger records every error so a test can assert it was logged.
type TestLogger struct {
	mu     sync.Mutex
	errors []LogEntry
}

// NewTestLogger returns an empty TestLogger.
func NewTestLogger() *TestLogger { return &TestLogger{} }

func (l *TestLogger) Info(context.Context, string, ...Field) {}

func (l *TestLogger) Error(_ context.Context, msg string, err error, fields ...Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	l.errors = append(l.errors, LogEntry{Message: msg, Error: err, Fields: m})
}

// LastError is the most recent error logged, or nil when none was.
func (l *TestLogger) LastError() *LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.errors) == 0 {
		return nil
	}
	e := l.errors[len(l.errors)-1]
	return &e
}
