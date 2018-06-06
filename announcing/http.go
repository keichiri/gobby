package announcing

import (
	"fmt"
	"gobby"
	"gobby/bencoding"
	"gobby/logs"
	"gobby/stats"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	_HTTP_TIMEOUT = time.Second * 5
)

func setupClient() *http.Client {
	transport := &http.Transport{}
	return &http.Client{
		Transport: transport,
		Timeout:   _HTTP_TIMEOUT,
	}
}

type httpAnnouncer struct {
	url          string
	stats        *stats.Stats
	downloadInfo *gobby.DownloadInfo
	client       *http.Client
	trackerID    string
	signalCh     chan bool
}

func NewHTTPAnnouncer(url string, info *gobby.DownloadInfo) *httpAnnouncer {
	return &httpAnnouncer{
		url:          url,
		downloadInfo: info,
		client:       setupClient(),
		signalCh:     make(chan bool),
	}
}

func (a *httpAnnouncer) Stop() {
	a.signalCh <- true
}

func (a *httpAnnouncer) AnnounceCompletion() {
	a.signalCh <- false
}

// TODO - rethink this API
// Might make more sense to have a reference on a Coordinator and interface
// via method call instead of having a channel
func (a *httpAnnouncer) Announcing(resultCh chan<- *AnnounceResult) error {
	var res *AnnounceResult
	var err error
	var interval int

	logs.Debug("Announcer", "Starting announcer to %s", a.url)
	res, interval, err = a.announce("started")
	if err != nil {
		close(resultCh)
		return fmt.Errorf("Failed to announce started: %s", err)
	}

	resultCh <- res

	for {
		select {
		case <-time.After(time.Second * time.Duration(interval)):
			logs.Debug("Announcer", "Announcing regularly to %s", a.url)
			res, interval, err = a.announce("")
			if err != nil {
				close(resultCh)
				return fmt.Errorf("Failed to announce regularly: %s", err)
			}

			resultCh <- res
		case signal := <-a.signalCh:
			// signifies stop
			if signal {
				logs.Debug("Announcer", "Announcing stopped to %s", a.url)
				a.announce("stopped")
				close(resultCh)
				break
			} else {
				logs.Debug("Announcer", "Announcing completed to %s", a.url)
				res, interval, err = a.announce("completed")
				if err != nil {
					close(resultCh)
					return fmt.Errorf("Failed to announce completed: %s", err)
				}

				resultCh <- res
			}
		}
	}

	return nil
}

func (a *httpAnnouncer) announce(event string) (*AnnounceResult, int, error) {
	params := a.collectParams(event)
	queryString := params.Encode()
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
	return a.readTrackerResponse(decodedResponse)
}

func (a *httpAnnouncer) readTrackerResponse(response map[string]interface{}) (*AnnounceResult, int, error) {
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

func (a *httpAnnouncer) collectParams(event string) *url.Values {
	params := &url.Values{}
	currentStats := a.stats.GetCurrent()

	params.Set("info_hash", string(a.downloadInfo.InfoHash))
	params.Set("peer_id", string(a.downloadInfo.PeerID))
	params.Set("port", a.downloadInfo.Port)
	if event != "" {
		params.Set("event", event)
	}
	params.Set("downloaded", string(currentStats.Downloaded))
	params.Set("uploaded", string(currentStats.Uploaded))
	params.Set("left", string(currentStats.Left))
	params.Set("numwant", "20") // TODO
	params.Set("compact", "1")
	if a.trackerID != "" {
		params.Set("trackerid", a.trackerID)
	}

	return params
}
