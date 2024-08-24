package storage

import "enssat.tv/autovodsaver/twitch"

type Storager interface {
	Save(filePath string) error
	GetVideos() ([]twitch.Video, error)
}
