package main

import (
	"time"

	"enssat.tv/autovodsaver/utils"
	"enssat.tv/autovodsaver/watchdog"
	"github.com/rs/zerolog"
)

const accessKey = ""
const secretKey = ""
const videoID = "2224928673"
const outputFile = "./danyetraz.mp4"

func main() {
	ctx := utils.GetDefaultContext()
	logger := ctx.Value(utils.LoggerKey).(*zerolog.Logger)

	logger.Info().Msg("AutoVODSaver (by EnssaTV)")
	/*
		store, newS3Error := storage.NewS3StorageWithContext(ctx, "http://51.159.98.189:9000", "eu-west", "enssatv", storage.Credentials{
			AccessKey: accessKey,
			SecretKey: secretKey,
			Session:   "",
		})
		if newS3Error != nil {
			logger.Error().Msg(newS3Error.Error())
			os.Exit(1)
		}

		video := twitch.GetVideoWithContext(ctx, videoID)
		if err := store.Save(&video, outputFile); err != nil {
			logger.Error().Msg(err.Error())
			os.Exit(1)
		}
	*/
	wd := watchdog.NewWithContext(ctx, "sqlite", "mistermv")
	ch := wd.OnVideoUpdate()
	go func() {
		for msg := range *ch {
			logger.Info().Msgf("watchdog said vod %s has status %s", msg.Id, msg.Status)
		}
	}()
	if err := wd.Run(); err != nil {
		logger.Error().Msg(err.Error())
	}
	time.Sleep(time.Second * 30)
	wd.Stop()
}
