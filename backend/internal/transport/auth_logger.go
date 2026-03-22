package transport

import (
	"fmt"

	"go.uber.org/zap"
)

// zapAuthLogger adapts go.uber.org/zap to the authkit.Logger interface.
type zapAuthLogger struct {
	log *zap.Logger
}

// NewZapAuthLogger returns a zapAuthLogger wrapping the given zap.Logger.
func NewZapAuthLogger(log *zap.Logger) *zapAuthLogger {
	return &zapAuthLogger{log: log}
}

func (l *zapAuthLogger) Info(msg string, args ...any) {
	l.log.Info(fmt.Sprintf(msg, args...))
}

func (l *zapAuthLogger) Error(msg string, args ...any) {
	l.log.Error(fmt.Sprintf(msg, args...))
}
