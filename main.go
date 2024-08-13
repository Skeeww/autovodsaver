package main

import (
	"context"
	"fmt"

	"enssat.tv/autovodsaver/twitch"
)

func main() {
	ctx := context.Background()
	video := twitch.GetVideoWithContext(ctx, "2220004521")
	playlist := video.GetPlaylist()
	fmt.Printf("found url: %s (resolution=%s;framerate=%f)\n", playlist.Url, playlist.Resolution, playlist.Framerate)
	video.GetChunks(playlist)
}
