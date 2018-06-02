package gobby

import (
	"bytes"
	"testing"
)

func TestMetafile(t *testing.T) {
	piecesInput := "6:pieces40:111111111111111111112222222222222222222212:piece lengthi50e"
	fileInput := "4:name8:test.txt6:lengthi99e"
	input := "d8:announce12:www.test.com4:infod" + piecesInput + fileInput + "ee"

	metafile, err := DecodeMetafile([]byte(input))
	if err != nil {
		t.Fatalf("Error while decoding metafile: %s", err)
	}

	if metafile.AnnounceURL != "www.test.com" {
		t.Fatalf("Bad announce url: %s", metafile.AnnounceURL)
	}

	if len(metafile.InfoHash) != 20 {
		t.Fatalf("Bad metafile info hash length")
	}

	// file testing
	if len(metafile.Files) != 1 {
		t.Fatalf("Expected 1 file. Got: %d", len(metafile.Files))
	}
	if metafile.Files[0].Length != 99 {
		t.Fatalf("Expected file length 99. Got: %d", metafile.Files[0].Length)
	}
	if metafile.Files[0].Path != "test.txt" {
		t.Fatalf("Expected path test.txt. Got: %s", metafile.Files[0].Path)
	}

	// pieces testing
	if len(metafile.Pieces) != 2 {
		t.Fatalf("Expected 2 pieces. Got: %d", len(metafile.Pieces))
	}

	piece0 := metafile.Pieces[0]
	if piece0.Length != 50 {
		t.Fatalf("Expected piece0 length 50. Got: %d", piece0.Length)
	}
	expectedPiece0Hash := []byte("11111111111111111111")
	if !bytes.Equal(piece0.Hash, expectedPiece0Hash) {
		t.Fatalf("Expected piece hash: %v. Got: %v", expectedPiece0Hash, piece0.Hash)
	}

	piece1 := metafile.Pieces[1]
	if piece1.Length != 49 {
		t.Fatalf("Expected piece1 length 49. Got: %d", piece1.Length)
	}
	expectedPiece1Hash := []byte("22222222222222222222")
	if !bytes.Equal(piece1.Hash, expectedPiece1Hash) {
		t.Fatalf("Expected piece hash: %v. Got: %v", expectedPiece1Hash, piece1.Hash)
	}

}
