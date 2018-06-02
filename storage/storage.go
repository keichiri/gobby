package storage

import (
	"errors"
	"fmt"
	"gobby"
	"gobby/logs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type DirectoryHandler struct {
	path             string
	piecesPath       string
	requestsMx       sync.Mutex
	cache            *cache
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
		cache:            newCache(30),
		retrieveRequests: make(map[int][]chan<- *RetrieveResult),
	}
	return handler, nil
}

func (dh *DirectoryHandler) StorePiece(index int, data []byte, resCh chan<- *StoreResult) {
	go func() {
		err := storePiece(dh.piecesPath, index, data)
		if err != nil {
			dh.cache.Put(index, data)
		}
		resCh <- &StoreResult{
			Index: index,
			Err:   err,
		}
	}()
}

func (dh *DirectoryHandler) RetrievePiece(index int, resCh chan<- *RetrieveResult) {
	data := dh.cache.Get(index)
	if data != nil {
		res := &RetrieveResult{
			Index: index,
			Data:  data,
			Err:   nil,
		}
		resCh <- res
		return
	}

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
	} else {
		dh.cache.Put(index, pieceData)
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

// Blocking operation, intended to be executed once per download process
// Assumes clean directory
// TODO - support continuation between different invocations and preparation if needed
func (dh *DirectoryHandler) ComposeFiles(fileInfos []*gobby.File) error {
	indexes, err := listExistingIndexes(dh.piecesPath)
	if err != nil {
		return fmt.Errorf("Failed to list existing indexes: %s", err)
	}

	for i := 0; i < len(indexes)-1; i++ {
		if indexes[i+1]-indexes[i] != 1 {
			return errors.New("Cannot compose files yet. Missing indexes")
		}
	}

	err = dh.populateFilesFromIndexes(fileInfos, indexes)
	if err != nil {
		cleanupFiles(dh.path)
		return fmt.Errorf("Failed to compose files: %s", err)
	}

	return nil
}

func (dh *DirectoryHandler) populateFilesFromIndexes(fileInfos []*gobby.File, indexes []int) error {
	var currentPieceIndex int
	var currentPieceOffset int

	for _, fileInfo := range fileInfos {
		fullPath := filepath.Join(dh.path, fileInfo.Path)
		directory := filepath.Dir(fullPath)
		err := os.MkdirAll(directory, 0700)
		if err != nil {
			return fmt.Errorf("Failed to create directory at path %s. Error: %s", directory, err)
		}

		f, err := os.Create(fullPath)
		if err != nil {
			return fmt.Errorf("Failed to create file at path: %s. Error: %s", fullPath, err)
		}

		toWrite := fileInfo.Length

		for toWrite > 0 {
			if currentPieceIndex >= len(indexes) {
				return errors.New("Failed to populate all files based on available pieces")
			}

			piecePath := createPiecePath(dh.piecesPath, currentPieceIndex)
			pieceFile, err := os.Open(piecePath)
			if err != nil {
				return fmt.Errorf("Failed to open piece file: %s", err)
			}

			pieceData, err := ioutil.ReadAll(pieceFile)
			if err != nil {
				return fmt.Errorf("Failed to read piece data: %s", err)
			}
			pieceFile.Close()

			pieceData = pieceData[currentPieceOffset:]

			if len(pieceData) <= toWrite {
				_, err = f.Write(pieceData)
				if err != nil {
					return fmt.Errorf("Failed to write to file: %s", err)
				}
				toWrite -= len(pieceData)
				currentPieceIndex += 1
				currentPieceOffset = 0
			} else {
				_, err = f.Write(pieceData[:toWrite])
				if err != nil {
					return fmt.Errorf("Failed to write to file: %s", err)
				}
				currentPieceOffset += toWrite
				toWrite = 0
			}
		}

		logs.Info("Storage", "Created %s", fullPath)
		f.Close()
	}

	return nil
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

func listExistingIndexes(piecesDirPath string) ([]int, error) {
	stats, err := ioutil.ReadDir(piecesDirPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to stat pieces directory: %s", err)
	}

	indexes := make([]int, 0)
	for _, stat := range stats {
		if !stat.IsDir() {
			name := stat.Name()
			if strings.HasSuffix(name, ".piece") {
				name = strings.TrimSuffix(name, ".piece")
				index, err := strconv.Atoi(name)
				if err == nil {
					indexes = append(indexes, index)
				}
			}
		}
	}

	sort.SliceStable(indexes, func(i int, j int) bool {
		return indexes[i] <= indexes[j]
	})

	return indexes, nil
}

func cleanupFiles(path string) {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		logs.Error("Storage", "Failed to stat %s for cleanup. Error: %s", path, err)
	}

	for _, info := range infos {
		if !info.IsDir() {
			fullPath := filepath.Join(path, info.Name())
			err := os.Remove(fullPath)
			if err != nil {
				logs.Error("Storage", "Failed to delete %s. Error: %s", fullPath, err)
			}
		} else {
			if info.Name() == ".pieces" {
				continue
			}

			dirPath := filepath.Join(path, info.Name())
			err := os.RemoveAll(dirPath)
			if err != nil {
				logs.Error("Storage", "Failed to delete %s. Error: %s", dirPath, err)
			}
		}
	}
}
