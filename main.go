package main

import (
	"context"
	"fmt"
	"os"

	"enssat.tv/autovodsaver/twitch"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const accessKey = "===REDACTED FOR OBIVIOUS REASONS (nvm ima going to put this in env)==="
const secretKey = "same story here, checkout this dude on apex"
const videoID = "2224928673"
const outputFile = "./test.mp4"

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	log.Logger = log.Level(zerolog.DebugLevel)

	log.Info().Msgf("Getting information from video id %s", videoID)
	video := twitch.GetVideo(videoID)
	log.Info().Msgf("Retrieving video name %s", video.Title)
	if err := video.Download(outputFile); err != nil {
		log.Error().Err(err)
	}
	log.Info().Msgf("Outpufile %s", outputFile)

	log.Info().Msg("Uploading the file to S3...")
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String("http://you really though i was going to show that my minio instance ip is 89.314."),
		Region:       "us-west-rack-2",
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	})
	file, _ := os.OpenFile(outputFile, os.O_RDONLY, 0660)
	_, err := client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String("enssatv"),
		Key:         aws.String(fmt.Sprintf("%s_%s", video.Title, video.Id)),
		Body:        file,
		ContentType: aws.String("video/mp4"),
	})
	if err != nil {
		panic(err)
	}
	log.Info().Msg("Upload done")
}
