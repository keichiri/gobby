package storage

import (
	"fmt"
	"gobby/logs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type DirectoryHandler struct {
	path             string
	piecesPath       string
	requestsMx       sync.Mutex
	retrieveRequests map[int][]chan<- *RetrieveResult
}

func NewDirectoryHandler(path string) (*DirectoryHandler, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to check permissions for path: %s", err)
	}
	if (stat.Mode() & 0700) != 0700 {
		return nil, fmt.Errorf("Insufficient permissions: %s", stat.Mode().String())
	}

	piecesPath := filepath.Join(path, "pieces")
	if _, err := os.Stat(piecesPath); err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(piecesPath, 0700)
			if err != nil {
				return nil, fmt.Errorf("Failed to create pieces path: %s", err)
			}
		} else {
			return nil, fmt.Errorf("Failed to check pieces path: %s", err)
		}
	}

	handler := &DirectoryHandler{
		path:             path,
		piecesPath:       piecesPath,
		retrieveRequests: make(map[int][]chan<- *RetrieveResult),
	}
	return handler, nil
}

func (dh *DirectoryHandler) StorePiece(index int, data []byte, resCh chan<- *StoreResult) {
	go func() {
		err := storePiece(dh.piecesPath, index, data)
		resCh <- &StoreResult{
			Index: index,
			Err:   err,
		}
	}()
}

func (dh *DirectoryHandler) RetrievePiece(index int, resCh chan<- *RetrieveResult) {
	dh.requestsMx.Lock()
	channels, exists := dh.retrieveRequests[index]
	if exists {
		channels = append(channels, resCh)
		dh.retrieveRequests[index] = channels
	} else {
		channels = []chan<- *RetrieveResult{resCh}
		dh.retrieveRequests[index] = channels
		go dh.retrieveAndSendPiece(index)
	}
	dh.requestsMx.Unlock()
}

func (dh *DirectoryHandler) retrieveAndSendPiece(index int) {
	pieceData, err := retrievePiece(dh.piecesPath, index)
	if err != nil {
		logs.Error("Storage", "Failed to retrieve piece %d. Error: %s", index, err)
	}

	dh.requestsMx.Lock()
	channels := dh.retrieveRequests[index]
	// Safe to be shared
	res := &RetrieveResult{
		Index: index,
		Data:  pieceData,
		Err:   err,
	}
	for _, channel := range channels {
		channel <- res
	}
	delete(dh.retrieveRequests, index)
	dh.requestsMx.Unlock()
}

type StoreResult struct {
	Index int
	Err   error
}

type RetrieveResult struct {
	Index int
	Data  []byte
	Err   error
}

func storePiece(piecesDirPath string, pieceIndex int, pieceData []byte) error {
	piecePath := createPiecePath(piecesDirPath, pieceIndex)
	f, err := os.Create(piecePath)
	if err != nil {
		return fmt.Errorf("Failed to create piece file: %s", err)
	}
	defer f.Close()

	_, err = f.Write(pieceData)
	if err != nil {
		return fmt.Errorf("Failed to write to piece file: %s", err)
	}

	return nil
}

func retrievePiece(piecesDirPath string, pieceIndex int) ([]byte, error) {
	piecePath := createPiecePath(piecesDirPath, pieceIndex)
	f, err := os.Open(piecePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open piece file: %s", err)
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("Failed to read piece file: %s", err)
	}

	return data, nil
}

func createPiecePath(piecesDirPath string, pieceIndex int) string {
	return filepath.Join(piecesDirPath, fmt.Sprintf("%d.piece", pieceIndex))
}
