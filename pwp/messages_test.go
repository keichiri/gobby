package pwp

import (
	"bytes"
	"testing"
)

func TestEncodingDecoding(t *testing.T) {
	messages := []Message{
		&BitfieldMsg{[]byte("this is a test bitfield")},
		&KeepAliveMsg{},
		&HaveMsg{500000},
		&UnchokeMsg{},
		&InterestedMsg{},
		&KeepAliveMsg{},
		&RequestMsg{10, 20, 10000},
		&ChokeMsg{},
		&CancelMsg{0, 1000000, 10000},
		&KeepAliveMsg{},
		&PieceMsg{10, 20, make([]byte, 1000, 1000)},
		&UninterestedMsg{},
	}

	encodedMessages := make([][]byte, 0)
	for _, msg := range messages {
		encodedMessages = append(encodedMessages, msg.Encode())
	}

	data := bytes.Join(encodedMessages, nil)
	data = append(data, []byte("leftover")...)

	decodedMessages, leftover, err := DecodeMessages(data)
	if err != nil {
		t.Fatalf("Failed to decode messages: %s", err)
	}
	if len(messages) != len(decodedMessages) {
		t.Fatalf("Length missmatch. Expected: %d. Got: %d", len(messages), len(decodedMessages))
	}
	if string(leftover) != "leftover" {
		t.Fatalf("Invalid leftover: %v", leftover)
	}
	msg1 := decodedMessages[0].(*BitfieldMsg)
	if string(msg1.Bitfield) != "this is a test bitfield" {
		t.Fatalf("Wrong bitfield: %v", msg1.Bitfield)
	}
	_ = decodedMessages[1].(*KeepAliveMsg)
	msg3 := decodedMessages[2].(*HaveMsg)
	if msg3.Index != 500000 {
		t.Fatalf("Wrong index: %d", msg3.Index)
	}
	_ = decodedMessages[3].(*UnchokeMsg)
	_ = decodedMessages[4].(*InterestedMsg)
	_ = decodedMessages[5].(*KeepAliveMsg)
	msg7 := decodedMessages[6].(*RequestMsg)
	if msg7.Index != 10 {
		t.Fatalf("Wrong index: %d", msg7.Index)
	}
	if msg7.Offset != 20 {
		t.Fatalf("Wrong offset: %d", msg7.Offset)
	}
	if msg7.Length != 10000 {
		t.Fatalf("Wrong length: %d", msg7.Length)
	}
	_ = decodedMessages[7].(*ChokeMsg)
	msg9 := decodedMessages[8].(*CancelMsg)
	if msg9.Index != 0 {
		t.Fatalf("Wrong index: %d", msg9.Index)
	}
	if msg9.Offset != 1000000 {
		t.Fatalf("Wrong offset: %d", msg9.Offset)
	}
	if msg9.Length != 10000 {
		t.Fatalf("Wrong length: %d", msg9.Length)
	}
	_ = decodedMessages[9].(*KeepAliveMsg)
	msg11 := decodedMessages[10].(*PieceMsg)
	if msg11.Index != 10 {
		t.Fatalf("Wrong index: %d", msg11.Index)
	}
	if msg11.Offset != 20 {
		t.Fatalf("Wrong offset: %d", msg11.Offset)
	}
	if !bytes.Equal(msg11.Block, make([]byte, 1000, 1000)) {
		t.Fatalf("Wrong block: %v", msg11.Block)
	}
	_ = decodedMessages[11].(*UninterestedMsg)
}

func TestEncodingIfInvalidMessage(t *testing.T) {
	messages := []Message{
		&BitfieldMsg{[]byte("this is a test bitfield")},
		&KeepAliveMsg{},
		&HaveMsg{500000},
		&UnchokeMsg{},
		&InterestedMsg{},
		&KeepAliveMsg{},
		&RequestMsg{10, 20, 10000},
		&ChokeMsg{},
		&CancelMsg{0, 1000000, 10000},
		&KeepAliveMsg{},
		&PieceMsg{10, 20, make([]byte, 1000, 1000)},
		&UninterestedMsg{},
	}

	encodedMessages := make([][]byte, 0)
	for _, msg := range messages {
		encodedMessages = append(encodedMessages, msg.Encode())
	}
	encodedMessages[5] = []byte{0, 0, 0, 5, 9, 1, 1, 1, 1}

	data := bytes.Join(encodedMessages, nil)
	data = append(data, []byte("leftover")...)

	_, _, err := DecodeMessages(data)
	if err == nil {
		t.Fatalf("Expected error")
	}
}
