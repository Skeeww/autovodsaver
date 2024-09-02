package watchdog

import (
	"context"
	"database/sql"
	"time"

	"enssat.tv/autovodsaver/constants"
	"enssat.tv/autovodsaver/queue"
	"enssat.tv/autovodsaver/twitch"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
)

const (
	refreshInterval                  = 15
	createVideosStatusTableStatement = `
		CREATE TABLE IF NOT EXISTS videos_status (
			id VARCHAR(50) NOT NULL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			description TEXT NOT NULL,
			published_at DATETIME NOT NULL,
			duration INTEGER NOT NULL,
			status VARCHAR(50) NOT NULL
		);
	`
)

type SQLiteWatchdog struct {
	Watchdog
	Database         *sql.DB
	DatabaseFilePath string
}

func NewSQLiteWatchdog(channelId string) *SQLiteWatchdog {
	return NewSQLiteWatchdogWithContext(context.Background(), channelId)
}

func NewSQLiteWatchdogWithContext(ctx context.Context, channelId string) *SQLiteWatchdog {
	ch := make(chan UpdateMessage, 100)
	return &SQLiteWatchdog{
		DatabaseFilePath: "./db.sqlite",
		Watchdog: Watchdog{
			Context:              ctx,
			Status:               WatchdogStatusStop,
			ChannelId:            channelId,
			OnVideoUpdateChannel: &ch,
			Queues: struct{ DownloadQueue *queue.FifoQueue }{
				DownloadQueue: queue.NewFifo(),
			},
		},
	}
}

func (wd *SQLiteWatchdog) Run() error {
	// Open connection to the database
	db, openDatabaseError := sql.Open("sqlite3", wd.DatabaseFilePath)
	if openDatabaseError != nil {
		return openDatabaseError
	}

	// Create table "videos_status"
	_, createVideoTableError := db.Exec(createVideosStatusTableStatement)
	if createVideoTableError != nil {
		return createVideoTableError
	}

	wd.Status = WatchdogStatusRun
	wd.Database = db
	go wd.watchdogTwitchVideos()
	go wd.watchdogDownloadQueue()
	return nil
}

func (wd *SQLiteWatchdog) Stop() error {
	wd.Status = WatchdogStatusStop
	if closeError := wd.Database.Close(); closeError != nil {
		return closeError
	}
	if wd.OnVideoUpdateChannel != nil {
		close(*wd.OnVideoUpdateChannel)
	}
	return nil
}

func (wd *SQLiteWatchdog) watchdogTwitchVideos() {
	logger := wd.Context.Value(constants.LoggerKey).(*zerolog.Logger)

	go func() {
		for msg := range *wd.OnVideoUpdateChannel {
			if msg.Status == constants.VideoStatusMissing {
				wd.updateVideoStatus(msg.Video, constants.VideoStatusQueued)
				wd.Queues.DownloadQueue.Enqueue(&msg.VideoWatched)
				logger.Info().Msgf("video %s added to download queue (size: %d)", msg.Id, wd.Queues.DownloadQueue.Size())
			}
		}
	}()

	for wd.Status == WatchdogStatusRun {
		logger.Info().Msgf("synchronizing videos from twitch channel %s", wd.ChannelId)
		if errSyncVideos := wd.syncVideos(); errSyncVideos != nil {
			logger.Error().Msg(errSyncVideos.Error())
		}
		time.Sleep(refreshInterval * time.Second)
	}
}

func (wd *SQLiteWatchdog) watchdogDownloadQueue() {
	logger := wd.Context.Value(constants.LoggerKey).(*zerolog.Logger)
	for video := wd.Queues.DownloadQueue.Dequeue(); true; {
		logger.Info().Msgf("video %s is being downloaded", video.Id)
		if downloadError := video.Download(video.Id); downloadError != nil {
			logger.Error().Msg(downloadError.Error())
			video.Status = constants.VideoStatusExpired
			wd.updateVideoStatus(video.Video, constants.VideoStatusExpired)
			continue
		}
		video.Status = constants.VideoStatusDownloaded
		wd.updateVideoStatus(video.Video, constants.VideoStatusDownloaded)
		logger.Info().Msgf("video %s has been downloaded", video.Id)
	}
}

func (wd *SQLiteWatchdog) syncVideos() error {
	logger := wd.Context.Value(constants.LoggerKey).(*zerolog.Logger)
	// List available vods
	vods := twitch.GetVideos(wd.ChannelId)
	logger.Debug().Msgf("found %d vods for the channel %s", len(vods), wd.ChannelId)

	// Get vods from local database
	dbVods, errGetVideos := wd.getVideos()
	if errGetVideos != nil {
		return errGetVideos
	}

	// Update database with missing vods
	for _, vod := range vods {
		// Search for the vod in local database
		exist := false
		for _, dbVod := range dbVods {
			if vod.Id == dbVod.Id {
				exist = true
				*wd.OnVideoUpdateChannel <- UpdateMessage{
					VideoWatched: dbVod,
				}
				continue
			}
		}
		// If the vod exists skip
		if exist {
			continue
		}
		// Else insert into the local database
		if addVideoError := wd.addVideo(vod); addVideoError != nil {
			logger.Error().Msg(addVideoError.Error())
			continue
		}
		logger.Debug().Msgf("add vod %s to database", vod.Id)
	}
	return nil
}

func (wd *SQLiteWatchdog) getVideos() ([]constants.VideoWatched, error) {
	rows, execError := wd.Database.QueryContext(wd.Context, "SELECT * FROM videos_status")
	if execError != nil {
		return nil, execError
	}
	defer rows.Close()

	videos := make([]constants.VideoWatched, 0)
	for rows.Next() {
		var (
			id           string
			title        string
			description  string
			published_at time.Time
			duration     uint
			status       constants.VideoStatus
		)
		rows.Scan(&id, &title, &description, &published_at, &duration, &status)
		videos = append(videos, constants.VideoWatched{
			Status: status,
			Video: twitch.Video{
				Context:       wd.Context,
				Id:            id,
				Title:         title,
				Description:   description,
				PublishedAt:   published_at,
				LengthSeconds: duration,
			},
		})
	}

	return videos, nil
}

func (wd *SQLiteWatchdog) addVideo(video twitch.Video) error {
	stmt, prepareError := wd.Database.PrepareContext(wd.Context, "INSERT INTO videos_status VALUES(?, ?, ?, ?, ?, ?)")
	if prepareError != nil {
		return prepareError
	}
	if _, execError := stmt.ExecContext(wd.Context, video.Id, video.Title, video.Description, video.PublishedAt, video.LengthSeconds, constants.VideoStatusMissing); execError != nil {
		return execError
	}
	*wd.OnVideoUpdateChannel <- UpdateMessage{
		VideoWatched: constants.VideoWatched{
			Video:  video,
			Status: constants.VideoStatusMissing,
		},
	}
	return nil
}

func (wd *SQLiteWatchdog) updateVideoStatus(video twitch.Video, newStatus constants.VideoStatus) error {
	// TODO: Cringe function, it should take an watchedVideo in parameter and change the instance status to avoid doing two things separatly
	stmt, prepareError := wd.Database.PrepareContext(wd.Context, "UPDATE videos_status SET status = ? WHERE id = ?")
	if prepareError != nil {
		return prepareError
	}
	if _, execError := stmt.ExecContext(wd.Context, newStatus, video.Id); execError != nil {
		return execError
	}
	*wd.OnVideoUpdateChannel <- UpdateMessage{
		VideoWatched: constants.VideoWatched{
			Video:  video,
			Status: newStatus,
		},
	}
	return nil
}
