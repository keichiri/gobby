package announcing

import (
	"fmt"
	"gobby/bencoding"
	"gobby/logs"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	_HTTP_TIMEOUT = time.Second * 5
)

type httpAdapter struct {
	url       string
	trackerID string
	client    *http.Client
}

func setupClient() *http.Client {
	transport := &http.Transport{}
	return &http.Client{
		Transport: transport,
		Timeout:   _HTTP_TIMEOUT,
	}
}

func newHTTPAdapter(url string) *httpAdapter {
	return &httpAdapter{
		url:    url,
		client: setupClient(),
	}
}

func (a *httpAdapter) Announce(params map[string]interface{}) (*AnnounceResult, int, error) {
	urlParams := a.processParams(params)
	queryString := urlParams.Encode()
	fullURL := a.url + "?" + queryString
	resp, err := a.client.Get(fullURL)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed HTTP request to tracker: %s", err)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to read HTTP response from tracker: %s", err)
	}

	_decodedResponse, err := bencoding.Decode(content)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to decode tracker response: %s", err)
	}
	decodedResponse, ok := _decodedResponse.(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("Invalid tracker response: %s", string(content))
	}

	logs.Debug("Announcer", "Raw tracker response from %s: %v", a.url, decodedResponse)
	return a.parseTrackerResponse(decodedResponse)
}

func (a *httpAdapter) processParams(params map[string]interface{}) *url.Values {
	urlValues := &url.Values{}
	urlValues.Set("info_hash", string(params["infoHash"].([]byte)))
	urlValues.Set("peer_id", string(params["peerID"].([]byte)))
	urlValues.Set("port", params["port"].(string))
	if event := params["event"].(string); event != "" {
		urlValues.Set("event", event)
	}

	urlValues.Set("downloaded", strconv.Itoa(params["downloaded"].(int)))
	urlValues.Set("uploaded", strconv.Itoa(params["uploaded"].(int)))
	urlValues.Set("left", strconv.Itoa(params["left"].(int)))
	urlValues.Set("compact", "1")
	urlValues.Set("numwant", strconv.Itoa(params["numwant"].(int)))

	if a.trackerID != "" {
		urlValues.Set("trackerid", a.trackerID)
	}

	return urlValues
}

func (a *httpAdapter) parseTrackerResponse(response map[string]interface{}) (*AnnounceResult, int, error) {
	_complete, exists := response["complete"]
	if !exists {
		return nil, 0, fmt.Errorf("Missing tracker response field: complete. Response: %v", response)
	}
	complete, ok := _complete.(int)
	if !ok {
		return nil, 0, fmt.Errorf("Invalid tracker response field: complete. Response: %v", response)
	}

	_incomplete, exists := response["incomplete"]
	if !exists {
		return nil, 0, fmt.Errorf("Missing tracker response field: incomplete. Response: %v", response)
	}
	incomplete, ok := _incomplete.(int)
	if !ok {
		return nil, 0, fmt.Errorf("Invalid tracker response field: incomplete. Response: %v", response)
	}

	_interval, exists := response["interval"]
	if !exists {
		return nil, 0, fmt.Errorf("Missing tracker response field: interval. Response: %v", response)
	}
	interval, ok := _interval.(int)
	if !ok {
		return nil, 0, fmt.Errorf("Invalid tracker response field: interval. Response: %v", response)
	}

	_peerData, exists := response["peers"]
	if !exists {
		return nil, 0, fmt.Errorf("Missing tracker response field: peers. Response: %v", response)
	}
	peerData, ok := _peerData.([]byte)
	if !ok {
		return nil, 0, fmt.Errorf("Invalid tracker response field: peers. Response: %v", response)
	}

	_trackerID, exists := response["tracker id"]
	if exists {
		trackerID, ok := _trackerID.([]byte)
		if !ok {
			return nil, 0, fmt.Errorf("Invalid tracker response field: tracker id. Response: %v", response)
		}

		a.trackerID = string(trackerID)
	}

	announceResult := &AnnounceResult{
		Complete:   complete,
		Incomplete: incomplete,
		PeerData:   peerData,
	}

	return announceResult, interval, nil
}
