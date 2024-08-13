package twitch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"enssat.tv/autovodsaver/twitch/internals"
	"github.com/grafov/m3u8"
)

type ContextKey int

const (
	videoPlaybackAccessToken ContextKey = iota
)

type Video struct {
	Context       context.Context
	Id            string    `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	PublishedAt   time.Time `json:"publishedAt"`
	LengthSeconds uint      `json:"lengthSeconds"`
}

type VideoPlaybackAccessToken struct {
	Value     string `json:"value"`
	Signature string `json:"signature"`
}

type PlaylistInfo struct {
	Url        string
	Resolution string
	Framerate  float64
	Chunked    bool
}

type Chunk struct {
	Id       string
	Duration float64
}

type videoResponse struct {
	Data struct {
		Video Video `json:"video"`
	} `json:"data"`
}

type tokenPlaybackResponse struct {
	Data struct {
		VideoPlaybackAccessToken VideoPlaybackAccessToken `json:"videoPlaybackAccessToken"`
	} `json:"data"`
}

func getVideoQuery(videoId string) string {
	return `{
		video(id: "` + videoId + `") {
			id
			title
			description
			publishedAt
			broadcastType
			lengthSeconds
		}
	}`
}

func getPlaybackTokenQuery(videoId string) string {
	return `{
		videoPlaybackAccessToken(
			id: "` + videoId + `",
            params: {
                platform: "web",
                playerBackend: "mediaplayer",
                playerType: "site"
            }
		) {
            value
			signature
        }
	}`
}

func parseM3U8(content string) *PlaylistInfo {
	buffer := bytes.NewBufferString(content)
	playlist, playlistType, err := m3u8.Decode(*buffer, true)
	if err != nil {
		panic(err)
	}
	if playlistType != m3u8.MASTER {
		return nil
	}
	p := playlist.(*m3u8.MasterPlaylist)
	// Extract the highest resolution master
	var master PlaylistInfo
	maxRes := 0
	for _, v := range p.Variants {
		if len(v.Resolution) == 0 {
			continue
		}
		dim := strings.Split(v.Resolution, "x")
		width, _ := strconv.Atoi(dim[0])
		height, _ := strconv.Atoi(dim[1])
		res := width * height
		if res > maxRes {
			master = PlaylistInfo{
				Url:        v.URI,
				Resolution: v.Resolution,
				Framerate:  v.FrameRate,
				Chunked:    false,
			}
			if v.Video == "chunked" {
				master.Chunked = true
			}
			maxRes = res
		}
	}
	return &master
}

func (v *Video) getPlaybackToken() VideoPlaybackAccessToken {
	tokens, err := internals.PostGraphQL[tokenPlaybackResponse](getPlaybackTokenQuery(v.Id))
	if err != nil {
		panic(err)
	}
	return tokens.Data.VideoPlaybackAccessToken
}

func GetVideo(videoId string) Video {
	return GetVideoWithContext(context.Background(), videoId)
}

func GetVideoWithContext(ctx context.Context, videoId string) Video {
	video, err := internals.PostGraphQL[videoResponse](getVideoQuery(videoId))
	if err != nil {
		panic(err)
	}
	video.Data.Video.Context = ctx
	return video.Data.Video
}

func (v *Video) GetPlaylist() *PlaylistInfo {
	value := v.Context.Value(videoPlaybackAccessToken)
	if value == nil {
		v.Context = context.WithValue(v.Context, videoPlaybackAccessToken, v.getPlaybackToken())
		return v.GetPlaylist()
	}
	tokens := value.(VideoPlaybackAccessToken)
	return parseM3U8(internals.GetPlaylists(v.Id, tokens.Value, tokens.Signature))
}

func (v *Video) GetChunks(playlist *PlaylistInfo) string {
	res, httpGetError := http.Get(playlist.Url)
	if httpGetError != nil {
		panic(httpGetError)
	}

	data, readAllError := io.ReadAll(res.Body)
	if readAllError != nil {
		panic(readAllError)
	}

	chunks, playlistType, decodeError := m3u8.Decode(*bytes.NewBuffer(data), true)
	if decodeError != nil {
		panic(decodeError)
	}
	if playlistType != m3u8.MEDIA {
		panic("wrong playlist type")
	}

	fmt.Println(chunks.(*m3u8.MediaPlaylist).Segments[0].URI)

	baseUrl := path.Dir(playlist.Url)
	fmt.Println(baseUrl)
	return baseUrl
}
