package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"

	"enssat.tv/autovodsaver/twitch"
)

func main() {
	ctx := context.Background()
	video := twitch.GetVideoWithContext(ctx, "2220004521")
	playlist := video.GetPlaylist()
	fmt.Printf("found url: %s (resolution=%s;framerate=%f)\n", playlist.Url, playlist.Resolution, playlist.Framerate)
	// chunks := video.GetChunks(playlist)

	tmpPath := path.Join(os.TempDir(), video.Id)
	if err := os.MkdirAll(tmpPath, os.ModeDir); err != nil {
		panic(err)
	}
	/*
		// Download all chunks
		for idx, c := range chunks {
			// Download a chunk
			filePath := path.Join(tmpPath, fmt.Sprintf("chunk_%d.ts", c.Id))
			f, _ := os.Create(filePath)
			content, _ := http.Get(c.Uri)
			io.Copy(f, content.Body)
			f.Close()
			fmt.Printf("%d/%d (%f%%)\t%s\n", idx, len(chunks), (float64(idx)/float64(len(chunks)))*100, filePath)
		}
	*/
	strA := "hello,"
	strB := " world!"
	os.WriteFile("./a", bytes.NewBufferString(strA).Bytes(), os.ModePerm)
	os.WriteFile("./b", bytes.NewBufferString(strB).Bytes(), os.ModePerm)
	cmd := exec.Command("type", "./a", "./b")
	file, _ := os.Create("./c")
	cmd.Stdout = file
	cmd.Run()
	file.Close()
}
