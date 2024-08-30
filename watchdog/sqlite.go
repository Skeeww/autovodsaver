package watchdog

import (
	"database/sql"
	"fmt"
	"time"

	"enssat.tv/autovodsaver/twitch"
	_ "github.com/mattn/go-sqlite3"
)

const (
	createVideosStatusTableStatement = `
		CREATE TABLE IF NOT EXISTS videos_status (
			id INTEGER NOT NULL PRIMARY KEY,
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
	return &SQLiteWatchdog{
		DatabaseFilePath: "./db.sqlite",
		Watchdog: Watchdog{
			Status:               WatchdogStatusStop,
			ChannelId:            channelId,
			VideoHashMap:         make(map[twitch.Video]VideoStatus),
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
	defer db.Close()

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

func (wd *SQLiteWatchdog) watchdog() {
	for wd.Status == WatchdogStatusRun {
		fmt.Println("hey")
		time.Sleep(1 * time.Second)
	}
}

func (wd *SQLiteWatchdog) Stop() error {
	wd.Status = WatchdogStatusStop
	return nil
}
