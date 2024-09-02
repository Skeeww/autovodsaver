package constants

import "enssat.tv/autovodsaver/twitch"

type contextKey int

const (
	LoggerKey contextKey = iota
	StorageKey
)

type VideoStatus string

const (
	VideoStatusQueued       = "VIDEO_STATUS_QUEUED"
	VideoStatusMissing      = "VIDEO_STATUS_MISSING"
	VideoStatusExpired      = "VIDEO_STATUS_EXPIRED"
	VideoStatusArchived     = "VIDEO_STATUS_ARCHIVED"
	VideoStatusDownloaded   = "VIDEO_STATUS_DOWNLOADED"
	VideoStatusConcatenated = "VIDEO_STATUS_CONCATENATED"
)

type VideoWatched struct {
	twitch.Video
	Status VideoStatus
}
