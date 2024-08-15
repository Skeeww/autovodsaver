package twitch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"enssat.tv/autovodsaver/twitch/internals"
	"github.com/grafov/m3u8"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type ContextKey int

const (
	videoPlaybackAccessToken ContextKey = iota
)

// Représente une VOD Twitch
type Video struct {
	Context       context.Context
	Id            string    `json:"id"`            // Identifiant de la vidéo
	Title         string    `json:"title"`         // Nom de la vidéo
	Description   string    `json:"description"`   // Description de la vidéo
	PublishedAt   time.Time `json:"publishedAt"`   // Date de publication
	LengthSeconds uint      `json:"lengthSeconds"` // Longueur de la vidéo (en secondes)
}

// Représente un token permettant d'accéder à la vidéo depuis le CDN de Twitch
type VideoPlaybackAccessToken struct {
	Value     string `json:"value"`     // Valeur du token
	Signature string `json:"signature"` // Signature du token
}

// Représente une playlist M3U8
type PlaylistInfo struct {
	Url        string  // Lien vers la playlist
	Resolution string  // Résolution des médias dans la playlist
	Framerate  float64 // Fréquences d'images des médias dans la playlist
	Chunked    bool    // Est-ce que la playlist contient plusieurs médias
}

// Représente un morceau de média dans une playlist M3U8
type Chunk struct {
	Id         uint64  // Numéro du morceau
	Uri        string  // Identifiant du morceau
	Duration   float64 // Durée du morceau (en secondes)
	Path       string  // Chemin d'accès au morceau
	Downloaded bool    // Indique si le morceau a été téléchargé
}

// Représente une réponse de l'api GraphQL de Twitch lors de la requête d'information sur une vidéo
type videoResponse struct {
	Data struct {
		Video Video `json:"video"`
	} `json:"data"`
}

// Représente une réponse de l'api GraphQL de Twitch lors de la requête d'un token d'accès sur une vidéo
type tokenPlaybackResponse struct {
	Data struct {
		VideoPlaybackAccessToken VideoPlaybackAccessToken `json:"videoPlaybackAccessToken"`
	} `json:"data"`
}

// Requête GraphQL pour récupéré les informations d'une vidéo à partir de son identifiant
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

// Requête GraphQL pour récupéré le token d'accès à une vidéo à partir de son identifiant
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

// Récupère la playlist M3U8 ayant la meilleur qualité
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

// Récupère le token d'accès de la vidéo
func (v *Video) getPlaybackToken() VideoPlaybackAccessToken {
	tokens, err := internals.PostGraphQL[tokenPlaybackResponse](getPlaybackTokenQuery(v.Id))
	if err != nil {
		panic(err)
	}
	return tokens.Data.VideoPlaybackAccessToken
}

// Récupère les informations de la vidéo à partir de son identifiant
func GetVideo(videoId string) Video {
	return GetVideoWithContext(context.Background(), videoId)
}

// Récupère les informations de la vidéo à partir de son identifiant, avec un contexte
func GetVideoWithContext(ctx context.Context, videoId string) Video {
	video, err := internals.PostGraphQL[videoResponse](getVideoQuery(videoId))
	if err != nil {
		panic(err)
	}
	video.Data.Video.Context = ctx
	return video.Data.Video
}

// Récupère la playlist de la vidéo
func (v *Video) GetPlaylist() *PlaylistInfo {
	value := v.Context.Value(videoPlaybackAccessToken)
	if value == nil {
		v.Context = context.WithValue(v.Context, videoPlaybackAccessToken, v.getPlaybackToken())
		return v.GetPlaylist()
	}
	tokens := value.(VideoPlaybackAccessToken)
	return parseM3U8(internals.GetPlaylists(v.Id, tokens.Value, tokens.Signature))
}

// Récupère les médias de la vidéo à partir d'une playlist
func (v *Video) GetChunks(playlist *PlaylistInfo) []Chunk {
	res, httpGetError := http.Get(playlist.Url)
	if httpGetError != nil {
		panic(httpGetError)
	}

	data, readAllError := io.ReadAll(res.Body)
	if readAllError != nil {
		panic(readAllError)
	}

	untypedMedia, playlistType, decodeError := m3u8.Decode(*bytes.NewBuffer(data), true)
	if decodeError != nil {
		panic(decodeError)
	}
	if playlistType != m3u8.MEDIA {
		return make([]Chunk, 0)
	}

	baseUrl := fmt.Sprintf("https://%s", path.Dir(strings.Replace(playlist.Url, "https://", "", 1)))
	medias := untypedMedia.(*m3u8.MediaPlaylist)
	chunks := make([]Chunk, 0)
	for _, s := range medias.GetAllSegments() {
		chunks = append(chunks, Chunk{
			Id:         s.SeqId,
			Uri:        fmt.Sprintf("%s/%s", baseUrl, s.URI),
			Duration:   s.Duration,
			Downloaded: false,
		})
	}

	return chunks
}

func Concatenate(chunks *[]Chunk, outputPath string) {
	bar := progressbar.NewOptions(len(*chunks),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("[cyan][2/2][reset] Concatenate chunks..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	if !isSorted(chunks) {
		panic("chunks are not in the right order")
	}
	file, outputFileError := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if outputFileError != nil {
		panic(outputFileError)
	}
	for _, chunk := range *chunks {
		chunkFile, chunkFileError := os.OpenFile(chunk.Path, os.O_RDONLY, os.ModePerm)
		if chunkFileError != nil {
			panic(chunkFileError)
		}
		_, copyError := io.Copy(file, chunkFile)
		if copyError != nil {
			panic(copyError)
		}
		chunkFile.Close()
		// fmt.Printf("%d/%d (%f%%)\t%d bytes copied\n", idx+1, len(*chunks), (float64(idx)/float64(len(*chunks)))*100, written)
		bar.Add(1)
	}
	file.Close()
}

func isSorted(chunks *[]Chunk) bool {
	for i := 1; i < len(*chunks); i++ {
		if (*chunks)[i].Id != (*chunks)[i-1].Id+1 {
			return false
		}
	}
	return true
}
