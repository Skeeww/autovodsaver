package watchdog

import "enssat.tv/autovodsaver/twitch"

const (
	VideoStatusMissing       = "VIDEO_STATUS_MISSING"
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
	Status               WatchdogStatus
	ChannelId            string
	VideoHashMap         map[twitch.Video]VideoStatus
	OnVideoUpdateChannel *chan UpdateMessage
}

type UpdateMessage struct {
	Type  VideoStatus
	Video twitch.Video
}

type Watchdoger interface {
	Run() error
	Stop() error
	OnVideoUpdate(ch *chan UpdateMessage)
}
