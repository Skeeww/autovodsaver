package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"enssat.tv/autovodsaver/twitch"
)

func main() {
	ctx := context.Background()
	video := twitch.GetVideoWithContext(ctx, "2212394181")
	playlist := video.GetPlaylist()
	fmt.Printf("found url: %s (resolution=%s;framerate=%f)\n", playlist.Url, playlist.Resolution, playlist.Framerate)
	chunks := video.GetChunks(playlist)

	tmpPath := path.Join(os.TempDir(), video.Id)
	if err := os.MkdirAll(tmpPath, os.ModeDir); err != nil {
		panic(err)
	}

	// Download all chunks
	for i := 0; i < len(chunks); i++ {
		// Download a chunk
		filePath := path.Join(tmpPath, fmt.Sprintf("chunk_%d.ts", chunks[i].Id))
		f, _ := os.Create(filePath)
		content, _ := http.Get(chunks[i].Uri)
		io.Copy(f, content.Body)
		f.Close()
		fmt.Printf("%d/%d (%f%%)\t%s\n", i+1, len(chunks), (float64(i)/float64(len(chunks)))*100, filePath)
		chunks[i].Path = filePath
		chunks[i].Downloaded = true
	}
	twitch.Concatenate(&chunks, "./vod.ts")
	os.Remove(tmpPath)
}
