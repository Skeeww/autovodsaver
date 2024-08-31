package watchdog

import (
	"context"
	"database/sql"
	"time"

	"enssat.tv/autovodsaver/twitch"
	"enssat.tv/autovodsaver/utils"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
)

const (
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
	return &SQLiteWatchdog{
		DatabaseFilePath: "./db.sqlite",
		Watchdog: Watchdog{
			Context:              ctx,
			Status:               WatchdogStatusStop,
			ChannelId:            channelId,
			OnVideoUpdateChannel: nil,
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
	go wd.watchdog()
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

func (wd *SQLiteWatchdog) OnVideoUpdate() *chan UpdateMessage {
	logger := wd.Context.Value(utils.LoggerKey).(*zerolog.Logger)
	if wd.OnVideoUpdateChannel != nil {
		logger.Debug().Msg("update channel already created, no new channel required")
		return wd.OnVideoUpdateChannel
	}
	ch := make(chan UpdateMessage)
	wd.OnVideoUpdateChannel = &ch
	logger.Debug().Msg("new update channel created")
	return wd.OnVideoUpdateChannel
}

func (wd *SQLiteWatchdog) watchdog() {
	logger := wd.Context.Value(utils.LoggerKey).(*zerolog.Logger)
	for wd.Status == WatchdogStatusRun {
		logger.Info().Msgf("synchronizing videos from twitch channel %s", wd.ChannelId)
		if errSyncVideos := wd.syncVideos(); errSyncVideos != nil {
			logger.Error().Msg(errSyncVideos.Error())
		}
		time.Sleep(15 * time.Second)
	}
}

func (wd *SQLiteWatchdog) syncVideos() error {
	logger := wd.Context.Value(utils.LoggerKey).(*zerolog.Logger)
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

func (wd *SQLiteWatchdog) getVideos() ([]VideoWatched, error) {
	rows, execError := wd.Database.QueryContext(wd.Context, "SELECT * FROM videos_status")
	if execError != nil {
		return nil, execError
	}
	defer rows.Close()

	videos := make([]VideoWatched, 0)
	for rows.Next() {
		var (
			id           string
			title        string
			description  string
			published_at time.Time
			duration     uint
			status       VideoStatus
		)
		rows.Scan(&id, &title, &description, &published_at, &duration, &status)
		videos = append(videos, VideoWatched{
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
	if _, execError := stmt.ExecContext(wd.Context, video.Id, video.Title, video.Description, video.PublishedAt, video.LengthSeconds, VideoStatusMissing); execError != nil {
		return execError
	}
	*wd.OnVideoUpdateChannel <- UpdateMessage{
		VideoWatched: VideoWatched{
			Video:  video,
			Status: VideoStatusMissing,
		},
	}
	return nil
}

func (wd *SQLiteWatchdog) updateVideoStatus(video twitch.Video, newStatus VideoStatus) error {
	stmt, prepareError := wd.Database.PrepareContext(wd.Context, "UPDATE videos_status SET status = ? WHERE id = ?")
	if prepareError != nil {
		return prepareError
	}
	if _, execError := stmt.ExecContext(wd.Context, newStatus, video.Id); execError != nil {
		return execError
	}
	*wd.OnVideoUpdateChannel <- UpdateMessage{
		VideoWatched: VideoWatched{
			Video:  video,
			Status: newStatus,
		},
	}
	return nil
}
