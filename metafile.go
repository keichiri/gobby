package gobby

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"gobby/bencoding"
	"path"
)

type Metafile struct {
	AnnounceURL string
	InfoHash    []byte
	Pieces      []*Piece
	Files       []*File
}

func DecodeMetafile(encoded []byte) (*Metafile, error) {
	_metafileMap, err := bencoding.Decode(encoded)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode metafile content: %s", err)
	}
	metafileMap, ok := _metafileMap.(map[string]interface{})
	if !ok {
		return nil, errors.New("Invalid metafile content")
	}

	_url, exists := metafileMap["announce"]
	if !exists {
		return nil, errors.New("Missing required field: announce")
	}
	url, ok := _url.([]byte)
	if !ok {
		return nil, errors.New("Invalid field: announce")
	}

	_info, exists := metafileMap["info"]
	if !exists {
		return nil, errors.New("Missing required field: info")
	}
	info, ok := _info.(map[string]interface{})
	if !ok {
		return nil, errors.New("Invalid field: info")
	}

	files, err := ParseFiles(info)
	if err != nil {
		return nil, err
	}
	pieces, err := ParsePieces(info)
	if err != nil {
		return nil, err
	}

	totalFileLength := 0
	for _, file := range files {
		totalFileLength += file.Length
	}
	pieces[len(pieces)-1].Length = totalFileLength % pieces[len(pieces)-1].Length

	encodedInfo, _ := bencoding.Encode(info)
	encodedInfoIndex := bytes.Index(encoded, []byte("4:info"))
	actualEncodedInfo := encoded[encodedInfoIndex+6 : encodedInfoIndex+6+len(encodedInfo)]
	hasher := sha1.New()
	hasher.Write(actualEncodedInfo)
	infoHash := hasher.Sum(nil)

	metafile := &Metafile{
		AnnounceURL: string(url),
		Pieces:      pieces,
		Files:       files,
		InfoHash:    infoHash,
	}
	return metafile, nil
}

type File struct {
	Path   string
	Length int
}

func ParseFiles(info map[string]interface{}) ([]*File, error) {
	_name, exists := info["name"]
	if !exists {
		return nil, errors.New("Missing required field: name")
	}
	nameBytes, ok := _name.([]byte)
	if !ok {
		return nil, errors.New("Invalid field: name")
	}
	name := string(nameBytes)

	_length, exists := info["length"]
	if exists {
		length, ok := _length.(int)
		if !ok {
			return nil, errors.New("Invalid field: length")
		}

		file := &File{
			Path:   name,
			Length: length,
		}
		return []*File{file}, nil
	} else {
		_fileInfos, exists := info["files"]
		if !exists {
			return nil, errors.New("Missing required field: files or length")
		}
		fileInfos, ok := _fileInfos.([]interface{})
		if !ok {
			return nil, errors.New("Invalid field: files")
		}

		files := make([]*File, 0, len(fileInfos))
		for _, _fileInfo := range fileInfos {
			fileInfo, ok := _fileInfo.(map[string]interface{})
			if !ok {
				return nil, errors.New("Invalid field: files")
			}

			_length, exists := fileInfo["length"]
			if !exists {
				return nil, errors.New("Invalid field: files")
			}
			length, ok := _length.(int)
			if !ok {
				return nil, errors.New("Invalid field: files")
			}

			_subpathComponents, exists := fileInfo["path"]
			if !exists {
				return nil, errors.New("Invalid field: files")
			}
			subpathComponents, ok := _subpathComponents.([]interface{})
			if !ok {
				return nil, errors.New("Invalid field: files")
			}

			pathPieces := []string{name}
			for _, _component := range subpathComponents {
				componentBytes, ok := _component.([]byte)
				if !ok {
					return nil, errors.New("Invalid field: files")
				}

				component := string(componentBytes)
				pathPieces = append(pathPieces, component)
			}

			file := &File{
				Length: length,
				Path:   path.Join(pathPieces...),
			}
			files = append(files, file)
		}

		return files, nil

	}
}

type Piece struct {
	Index  int
	Length int
	Hash   []byte
	Data   []byte
}

func ParsePieces(info map[string]interface{}) ([]*Piece, error) {
	_pieceLength, exists := info["piece length"]
	if !exists {
		return nil, errors.New("Missing required field: piece length")
	}
	pieceLength, ok := _pieceLength.(int)
	if !ok {
		return nil, errors.New("Invalid field: piece length")
	}

	_hashes, exists := info["pieces"]
	if !exists {
		return nil, errors.New("Missing required field: pieces")
	}
	hashes, ok := _hashes.([]byte)
	if !ok || len(hashes)%20 != 0 {
		return nil, errors.New("Invalid field: pieces")
	}

	pieceCount := len(hashes) / 20
	pieces := make([]*Piece, pieceCount)

	for i := 0; i < pieceCount; i++ {
		piece := &Piece{
			Index:  i,
			Length: pieceLength,
			Hash:   hashes[i*20 : (i+1)*20],
		}
		pieces[i] = piece
	}

	return pieces, nil
}
