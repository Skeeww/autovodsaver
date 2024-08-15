package main

import (
	"fmt"

	"enssat.tv/autovodsaver/twitch"
)

func main() {
	video := twitch.GetVideo("2184677598")
	if err := video.Download("./danyetraz.mp4"); err != nil {
		fmt.Println(err)
	}
}
