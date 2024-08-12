package internals

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const gqlEndpoint = "https://gql.twitch.tv/gql"
const clientId = "kd1unb4b3q4t58fwlpcbzcbnm76a8fp"

func PostGraphQL[T any](query string) (T, error) {
	client := &http.Client{}

	payload, jsonError := json.Marshal(map[string]string{
		"query": query,
	})
	if jsonError != nil {
		return *new(T), jsonError
	}

	reader := bytes.NewReader(payload)
	req, requestError := http.NewRequest(http.MethodPost, gqlEndpoint, reader)
	if requestError != nil {
		return *new(T), requestError
	}
	req.Header.Set("Client-Id", clientId)

	res, responseError := client.Do(req)
	if responseError != nil {
		return *new(T), responseError
	}
	if res.StatusCode != http.StatusOK {
		return *new(T), fmt.Errorf("post graphql failed with status code %d reason: %s", res.StatusCode, res.Status)
	}

	result, readAllError := io.ReadAll(res.Body)
	if readAllError != nil {
		return *new(T), readAllError
	}

	var data T
	if unjsonError := json.Unmarshal(result, &data); unjsonError != nil {
		return *new(T), unjsonError
	}
	return data, nil
}
