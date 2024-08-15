package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"sync"

	"enssat.tv/autovodsaver/twitch"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

func main() {
	ctx := context.Background()
	video := twitch.GetVideoWithContext(ctx, "2212394181")
	playlist := video.GetPlaylist()
	fmt.Printf("found url: %s (resolution=%s;framerate=%f)\n", playlist.Url, playlist.Resolution, playlist.Framerate)
	chunks := video.GetChunks(playlist)

	tmpPath := path.Join(os.TempDir(), video.Id)
	if err := os.MkdirAll(tmpPath, os.ModeDir|os.ModePerm); err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}

	bar := progressbar.NewOptions(len(chunks),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][1/2][reset] Downloading chunks..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	// Download all chunks
	for i := 0; i < len(chunks); i++ {
		wg.Add(1)
		go func(b *progressbar.ProgressBar) {
			// Download a chunk
			filePath := path.Join(tmpPath, fmt.Sprintf("chunk_%d.ts", chunks[i].Id))
			f, _ := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, os.ModePerm)
			content, _ := http.Get(chunks[i].Uri)
			io.Copy(f, content.Body)
			f.Close()
			// fmt.Printf("%d/%d (%f%%)\t%s\n", i+1, len(chunks), (float64(i)/float64(len(chunks)))*100, filePath)
			chunks[i].Path = filePath
			chunks[i].Downloaded = true
			b.Add(1)
			wg.Done()
		}(bar)
	}
	wg.Wait()
	fmt.Println("")

	twitch.Concatenate(&chunks, "./vod.ts")
	os.Remove(tmpPath)
}
