package watchdog

import (
	"context"

	"enssat.tv/autovodsaver/constants"
	"enssat.tv/autovodsaver/queue"
)

const (
	WatchdogStatusRun  = "WATCHDOG_STATUS_RUN"
	WatchdogStatusStop = "WATCHDOG_STATUS_STOP"
)

type WatchdogStatus string

type Watchdog struct {
	Context              context.Context
	Status               WatchdogStatus
	ChannelId            string
	OnVideoUpdateChannel *chan UpdateMessage
	Queues               struct {
		DownloadQueue *queue.FifoQueue
	}
}

type UpdateMessage struct {
	constants.VideoWatched
}

type Watchdoger interface {
	Run() error
	Stop() error
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
