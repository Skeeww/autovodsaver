package watchdog

import (
	"context"

	"enssat.tv/autovodsaver/twitch"
)

const (
	VideoStatusMissing       = "VIDEO_STATUS_MISSING"
	VideoStatusExpired       = "VIDEO_STATUS_EXPIRED"
	VideoStatusArchived      = "VIDEO_STATUS_ARCHIVED"
	VideoStatusDownloading   = "VIDEO_STATUS_DOWNLOADING"
	VideoStatusConcatenating = "VIDEO_STATUS_CONCATENATING"
)

const (
	WatchdogStatusRun  = "WATCHDOG_STATUS_RUN"
	WatchdogStatusStop = "WATCHDOG_STATUS_STOP"
)

type VideoStatus string
type WatchdogStatus string

type Watchdog struct {
	Context              context.Context
	Status               WatchdogStatus
	ChannelId            string
	OnVideoUpdateChannel *chan UpdateMessage
}

type VideoWatched struct {
	twitch.Video
	Status VideoStatus
}

type UpdateMessage struct {
	VideoWatched
}

type Watchdoger interface {
	Run() error
	Stop() error
	OnVideoUpdate() *chan UpdateMessage
}

func New(kind string, channelId string) Watchdoger {
	return NewWithContext(context.Background(), kind, channelId)
}

func NewWithContext(ctx context.Context, kind string, channelId string) Watchdoger {
	if kind == "sqlite" {
		return NewSQLiteWatchdogWithContext(ctx, channelId)
	}
	return nil
}
