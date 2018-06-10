package pwp

import (
	"bytes"
	"errors"
	"fmt"
)

func EncodeHandshake(infoHash, peerID []byte) []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(19)
	buf.WriteString("BitTorrent Protocol")
	for i := 0; i < 8; i++ {
		buf.WriteByte(0)
	}
	buf.Write(infoHash)
	buf.Write(peerID)
	return buf.Bytes()
}

func ParseHandshake(encoded []byte) ([]byte, []byte, error) {
	if len(encoded) != 68 {
		return nil, nil, fmt.Errorf("Invalid handshake length: %d", len(encoded))
	}

	if encoded[0] != 19 {
		return nil, nil, errors.New("Invalid protocol string length byte")
	}

	protocolString := string(encoded[1:20])
	if protocolString != "BitTorrent Protocol" {
		return nil, nil, fmt.Errorf("Invalid protocol string: %s", protocolString)
	}

	return encoded[28:48], encoded[48:68], nil
}
