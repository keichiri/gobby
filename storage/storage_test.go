package storage

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

const path = "/tmp/gobby/storage_tests/"

func createHandler() *DirectoryHandler {
	path := "/tmp/gobby/storage_tests/"
	os.RemoveAll(path)
	os.MkdirAll(path, 0700)
	handler, err := NewDirectoryHandler(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to create directoy handler: %s", err))
	}
	return handler
}

func TestStorePiece(t *testing.T) {
	handler := createHandler()
	index := 1
	resCh := make(chan *StoreResult, index)
	data := []byte("test_data")

	handler.StorePiece(1, data, resCh)
	res := <-resCh

	if res.Err != nil {
		t.Fatalf("Error occurred: %s", res.Err)
	}

	if res.Index != index {
		t.Fatalf("Expected index: %d. Got: %d", index, res.Index)
	}

	readData, err := ioutil.ReadFile(path + "/pieces/1.piece")
	if err != nil {
		t.Fatalf("Failed to read piece: %s", err)
	}

	if !bytes.Equal(data, readData) {
		t.Fatalf("Expected %s. Got %s", string(data), string(readData))
	}
}

func TestRetrievePiece(t *testing.T) {
	handler := createHandler()
	index := 1
	resCh := make(chan *RetrieveResult, 1)
	data := []byte("test_data")

	err := ioutil.WriteFile(path+"/pieces/1.piece", data, 0700)
	if err != nil {
		t.Fatalf("Failed to prepare piece: %s", err)
	}

	handler.RetrievePiece(index, resCh)
	res := <-resCh

	if res.Err != nil {
		t.Fatalf("Error occurred: %s", res.Err)
	}

	if res.Index != index {
		t.Fatalf("Expected index: %d. Got: %d", index, res.Index)
	}

	if !bytes.Equal(data, res.Data) {
		t.Fatalf("Expected %s. Got %s", string(data), string(res.Data))
	}
}

func BenchmarkRetrieve(b *testing.B) {
	// run the Fib function b.N times
	handler := createHandler()
	index := 1
	resCh := make(chan *RetrieveResult, 10)
	data := []byte("test_data")
	err := ioutil.WriteFile(path+"/pieces/1.piece", data, 0700)
	if err != nil {
		b.Fatalf("Failed to prepare piece: %s", err)
	}

	for n := 0; n < b.N; n++ {
		handler.RetrievePiece(index, resCh)
		<-resCh
	}
}
