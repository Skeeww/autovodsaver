package main

import (
	"context"
	"os"

	"enssat.tv/autovodsaver/constants"
	"enssat.tv/autovodsaver/storage"
	"enssat.tv/autovodsaver/watchdog"
	"github.com/rs/zerolog"
)

const (
	accessKey = ""
	secretKey = ""
)

func main() {
	ctx := context.Background()

	// Setup logger
	logger := zerolog.New(nil).Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.InfoLevel)
	ctx = context.WithValue(ctx, constants.LoggerKey, &logger)

	// Setup storage
	store, newS3Error := storage.NewS3StorageWithContext(ctx, "http://:9000", "eu-west", "enssatv", storage.Credentials{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Session:   "",
	})
	if newS3Error != nil {
		logger.Error().Msg(newS3Error.Error())
		os.Exit(1)
	}
	ctx = context.WithValue(ctx, constants.StorageKey, &store)

	logger.Info().Msg("AutoVODSaver (by EnssaTV)")
	wd := watchdog.NewWithContext(ctx, "sqlite", "mistermv")
	wd.Run()
	defer wd.Stop()
	select {}
}
