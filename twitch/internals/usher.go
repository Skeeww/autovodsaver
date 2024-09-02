package internals

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const httpEndpoint = "https://usher.ttvnw.net/vod"

func GetPlaylists(videoId string, accessToken string, signature string) (string, error) {
	params := url.Values{}
	params.Add("nauth", accessToken)
	params.Add("nauthsig", signature)
	params.Add("allow_audio_only", "true")
	params.Add("allow_source", "true")
	params.Add("player", "twitchweb")

	res, responseError := http.Get(fmt.Sprintf("%s/%s?%s", httpEndpoint, videoId, params.Encode()))
	if responseError != nil {
		return "", responseError
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wrong status code %s", res.Status)
	}
	payload, readAllError := io.ReadAll(res.Body)
	if readAllError != nil {
		return "", readAllError
	}

	return string(payload), nil
}
