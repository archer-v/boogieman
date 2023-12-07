package model

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

var DefaultLogger = log.New(os.Stdout, "boogieman ", log.LstdFlags)

type LoggerContextKeyType int

const (
	LoggerContextKey         LoggerContextKeyType = iota
	LoggerPrefixStartBracket                      = "["
	LoggerPrefixEndBracket                        = "]"
)

type Logger interface {
	Print(v ...any)
	Println(v ...any)
	Printf(format string, v ...any)
}

func GetLogger(ctx context.Context) Logger {
	logger, ok := ctx.Value(LoggerContextKey).(Logger)
	if !ok {
		return DefaultLogger
	}
	return logger
}

func ContextWithLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, LoggerContextKey, l)
}

type ChainLogger struct {
	Prefix string
	Logger Logger
}

func NewChainLogger(l Logger, prefix ...string) Logger {
	logger := ChainLogger{
		Logger: l,
	}
	for _, p := range prefix {
		logger.Prefix = prefixString(p, logger.Prefix)
	}
	return &logger
}

func (l ChainLogger) Print(v ...any) {
	// add leading whitespace if needed
	if len(v) > 0 {
		v[0] = l.fixLeadingWhitespace(v[0])
	}
	v = append([]any{l.Prefix}, v...)
	l.Logger.Print(v...)
}
func (l ChainLogger) Println(v ...any) {
	v = append(v, "\n")
	l.Print(v...)
}

func (l ChainLogger) Printf(format string, v ...any) {
	format = l.fixLeadingWhitespace(format).(string)
	l.Logger.Printf(l.Prefix+format, v...)
}

func prefixString(p string, s string) string {
	if p == "" {
		return s
	}
	return fmt.Sprintf("%v%v%v%v", LoggerPrefixStartBracket, p, LoggerPrefixEndBracket, s)
}

func (l ChainLogger) fixLeadingWhitespace(s any) any {
	if s, ok := s.(string); ok {
		if !strings.HasPrefix(s, " ") {
			if l.Prefix != "" &&
				!strings.HasSuffix(l.Prefix, " ") &&
				!strings.HasPrefix(s, LoggerPrefixStartBracket) {
				return " " + s
			}
		}
	}
	return s
}
