package utils

import (
	"context"
	"os"

	"enssat.tv/autovodsaver/storage"
	"github.com/rs/zerolog"
)

type contextKey int

const (
	LoggerKey contextKey = iota
	StorageKey
)

const accessKey = ""
const secretKey = ""

func GetDefaultContext() context.Context {
	ctx := context.Background()

	logger := zerolog.New(nil).Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.DebugLevel)
	ctx = context.WithValue(ctx, LoggerKey, &logger)

	store, newS3Error := storage.NewS3StorageWithContext(ctx, "http://51.159.98.189:9000", "eu-west", "enssatv", storage.Credentials{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Session:   "",
	})
	if newS3Error != nil {
		logger.Error().Msg(newS3Error.Error())
		os.Exit(1)
	}
	ctx = context.WithValue(ctx, StorageKey, &store)

	return ctx
}
