package utils

import (
	"context"
	"os"

	"github.com/rs/zerolog"
)

type contextKey int

const (
	LoggerKey contextKey = iota
)

func GetDefaultContext() context.Context {
	ctx := context.Background()

	logger := zerolog.New(nil).Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.DebugLevel)
	ctx = context.WithValue(ctx, LoggerKey, &logger)

	return ctx
}
