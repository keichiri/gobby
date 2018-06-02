package storage

import (
	"bytes"
	"fmt"
	"gobby"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const path = "/tmp/gobby/storage_tests/"

func createHandler() *DirectoryHandler {
	path := "/tmp/gobby/storage_tests/"
	os.RemoveAll(path)
	os.MkdirAll(path, 0700)
	handler, err := NewDirectoryHandler(path)
	if err != nil {
		panic(fmt.Sprintf("Failed to create directory handler: %s", err))
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

func TestComposeFiles(t *testing.T) {
	fileInfos := []*gobby.File{
		&gobby.File{Length: 7, Path: "dir0/file0.txt"},
		&gobby.File{Length: 5, Path: "/dir0/file1.txt"},
		&gobby.File{Length: 28, Path: "file2.txt"},
		&gobby.File{Length: 10, Path: "file3.txt"},
		&gobby.File{Length: 1, Path: "file4.txt"},
		&gobby.File{Length: 5, Path: "dir1/file5.txt"},
	}

	handler := createHandler()
	data0 := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	data1 := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	data2 := []byte{2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	data3 := []byte{3, 3, 3, 3, 3, 3, 3, 3, 3, 3}
	data4 := []byte{4, 4, 4, 4, 4, 4, 4, 4, 4, 4}
	data5 := []byte{5, 5, 5, 5, 5, 5}
	pieceDatas := [][]byte{data0, data1, data2, data3, data4, data5}

	for i, data := range pieceDatas {
		if err := ioutil.WriteFile(path+fmt.Sprintf("/pieces/%d.piece", i), data, 0700); err != nil {
			t.Fatalf("Failed to prepare piece: %s", err)
		}
	}

	err := handler.ComposeFiles(fileInfos)
	if err != nil {
		t.Fatalf("Failed to compose files: %s", err)
	}

	file0Content, err := ioutil.ReadFile(filepath.Join(path, "/dir0/file0.txt"))
	if err != nil {
		t.Fatalf("Failed to read file content. Err: %s", err)
	}
	expectedFile0Content := []byte{0, 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(file0Content, expectedFile0Content) {
		t.Fatalf("Expected file 0 content: %v. Got: %v", expectedFile0Content, file0Content)
	}

	file1Content, err := ioutil.ReadFile(filepath.Join(path, "/dir0/file1.txt"))
	if err != nil {
		t.Fatalf("Failed to read file content. Err: %s", err)
	}
	expectedFile1Content := []byte{0, 0, 0, 1, 1}
	if !bytes.Equal(file1Content, expectedFile1Content) {
		t.Fatalf("Expected file 1 content: %v. Got: %v", expectedFile1Content, file1Content)
	}

	file2Content, err := ioutil.ReadFile(filepath.Join(path, "file2.txt"))
	if err != nil {
		t.Fatalf("Failed to read file content. Err: %s", err)
	}
	expectedFile2Content := []byte{1, 1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}
	if !bytes.Equal(file2Content, expectedFile2Content) {
		t.Fatalf("Expected file 2 content: %v. Got: %v", expectedFile2Content, file2Content)
	}

	file3Content, err := ioutil.ReadFile(filepath.Join(path, "file3.txt"))
	if err != nil {
		t.Fatalf("Failed to read file content. Err: %s", err)
	}
	expectedFile3Content := []byte{4, 4, 4, 4, 4, 4, 4, 4, 4, 4}
	if !bytes.Equal(file3Content, expectedFile3Content) {
		t.Fatalf("Expected file 3 content: %v. Got: %v", expectedFile3Content, file3Content)
	}

	file4Content, err := ioutil.ReadFile(filepath.Join(path, "file4.txt"))
	if err != nil {
		t.Fatalf("Failed to read file content. Err: %s", err)
	}
	expectedFile4Content := []byte{5}
	if !bytes.Equal(file4Content, expectedFile4Content) {
		t.Fatalf("Expected file 4 content: %v. Got: %v", expectedFile4Content, file4Content)
	}

	file5Content, err := ioutil.ReadFile(filepath.Join(path, "dir1/file5.txt"))
	if err != nil {
		t.Fatalf("Failed to read file content. Err: %s", err)
	}
	expectedFile5Content := []byte{5, 5, 5, 5, 5}
	if !bytes.Equal(file5Content, expectedFile5Content) {
		t.Fatalf("Expected file 5 content: %v. Got: %v", expectedFile5Content, file5Content)
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
