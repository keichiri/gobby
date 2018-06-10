package announcing

import (
	"fmt"
	"gobby"
	"gobby/logs"
	"gobby/stats"
	"net/url"
	"time"
)

type announcer struct {
	url          string
	downloadInfo *gobby.DownloadInfo
	stats        *stats.Stats
	adapter      trackerAdapter
	signalCh     chan bool
}

func NewAnnouncer(trackerURL string, info *gobby.DownloadInfo, s *stats.Stats) (*announcer, error) {
	res, err := url.Parse(trackerURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse tracker URL: %s", err)
	}

	var adapter trackerAdapter
	if res.Scheme == "http" || res.Scheme == "https" {
		adapter = newHTTPAdapter(trackerURL)
	} else if res.Scheme == "udp" {
		adapter, err = newUDPAdapter(res.Hostname(), res.Port())
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("Unsupported tracker protocol: %s", res.Scheme)
	}

	announcer := &announcer{
		url:          trackerURL,
		downloadInfo: info,
		stats:        s,
		adapter:      adapter,
		signalCh:     make(chan bool, 2),
	}

	return announcer, nil
}

func (a *announcer) Stop() {
	a.signalCh <- true
}

func (a *announcer) AnnounceCompletion() {
	a.signalCh <- false
}

// TODO - rethink this API
// Might make more sense to have a reference on a Coordinator and interface
// via method call instead of having a channel
func (a *announcer) Announcing(resultCh chan<- *AnnounceResult) error {
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

func (a *announcer) announce(event string) (*AnnounceResult, int, error) {
	announceParams := a.prepareParams(event)
	return a.adapter.Announce(announceParams)
}

func (a *announcer) prepareParams(event string) map[string]interface{} {
	currentStats := a.stats.GetCurrent()

	return map[string]interface{}{
		"infoHash":   a.downloadInfo.InfoHash,
		"peerID":     a.downloadInfo.PeerID,
		"port":       a.downloadInfo.Port,
		"event":      event,
		"downloaded": currentStats.Downloaded,
		"uploaded":   currentStats.Uploaded,
		"left":       currentStats.Left,
		"numwant":    20,
	}
}
