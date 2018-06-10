package pwp

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type PWPMessage interface {
	Encode() []byte
}

type KeepAliveMsg struct{}

func (m *KeepAliveMsg) Encode() []byte {
	return []byte{0, 0, 0, 0}
}

type ChokeMsg struct{}

func (m *ChokeMsg) Encode() []byte {
	return []byte{0, 0, 0, 1, 0}
}

type UnchokeMsg struct{}

func (m *UnchokeMsg) Encode() []byte {
	return []byte{0, 0, 0, 1, 1}
}

type InterestedMsg struct{}

func (m *InterestedMsg) Encode() []byte {
	return []byte{0, 0, 0, 1, 2}
}

type UninterestedMsg struct{}

func (m *UninterestedMsg) Encode() []byte {
	return []byte{0, 0, 0, 1, 3}
}

type HaveMsg struct {
	Index int32
}

func (m *HaveMsg) Encode() []byte {
	b := []byte{0, 0, 0, 5, 4}
	buf := bytes.NewBuffer(b)
	binary.Write(buf, binary.BigEndian, m.Index)
	return buf.Bytes()
}

type BitfieldMsg struct {
	Bitfield []byte
}

func (m *BitfieldMsg) Encode() []byte {
	len := len(m.Bitfield) + 1
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, int32(len))
	buf.WriteByte(5)
	buf.Write(m.Bitfield)
	return buf.Bytes()
}

type RequestMsg struct {
	Index  int32
	Offset int32
	Length int32
}

func (m *RequestMsg) Encode() []byte {
	buf := bytes.NewBuffer([]byte{0, 0, 0, 13, 6})
	binary.Write(buf, binary.BigEndian, m.Index)
	binary.Write(buf, binary.BigEndian, m.Offset)
	binary.Write(buf, binary.BigEndian, m.Length)
	return buf.Bytes()
}

type PieceMsg struct {
	Index  int32
	Offset int32
	Block  []byte
}

func (m *PieceMsg) Encode() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, int32(len(m.Block)+9))
	buf.WriteByte(7)
	binary.Write(buf, binary.BigEndian, m.Index)
	binary.Write(buf, binary.BigEndian, m.Offset)
	buf.Write(m.Block)
	return buf.Bytes()
}

type CancelMsg struct {
	Index  int32
	Offset int32
	Length int32
}

func (m *CancelMsg) Encode() []byte {
	buf := bytes.NewBuffer([]byte{0, 0, 0, 13, 8})
	binary.Write(buf, binary.BigEndian, m.Index)
	binary.Write(buf, binary.BigEndian, m.Offset)
	binary.Write(buf, binary.BigEndian, m.Length)
	return buf.Bytes()
}

func DecodeMessages(data []byte) ([]PWPMessage, []byte, error) {
	messages := make([]PWPMessage, 0)

	for {
		if len(data) < 4 {
			break
		}

		messageLength := binary.BigEndian.Uint32(data[:4])
		if messageLength == 0 {
			messages = append(messages, &KeepAliveMsg{})
			data = data[4:]
			continue
		}

		if uint32(len(data[4:])) < messageLength {
			break
		}

		messageData := data[4 : 4+messageLength]
		msg, err := decodeMessage(messageData)
		if err != nil {
			return nil, nil, err
		}
		data = data[4+messageLength:]
		messages = append(messages, msg)
	}

	return messages, data, nil
}

func decodeMessage(data []byte) (PWPMessage, error) {
	switch data[0] {
	case 0:
		if len(data) > 1 {
			return nil, fmt.Errorf("Invalid choke message length: %d", len(data))
		}
		return &ChokeMsg{}, nil

	case 1:
		if len(data) > 1 {
			return nil, fmt.Errorf("Invalid unchoke message length: %d", len(data))
		}
		return &UnchokeMsg{}, nil

	case 2:
		if len(data) > 1 {
			return nil, fmt.Errorf("Invalid interested message length: %d", len(data))
		}
		return &InterestedMsg{}, nil

	case 3:
		if len(data) > 1 {
			return nil, fmt.Errorf("Invalid uninterested message length: %d", len(data))
		}
		return &UninterestedMsg{}, nil

	case 4:
		if len(data) != 5 {
			return nil, fmt.Errorf("Invalid have message length: %d", len(data))
		}
		index := int32(binary.BigEndian.Uint32(data[1:5]))
		msg := &HaveMsg{
			Index: index,
		}
		return msg, nil

	case 5:
		bitfield := make([]byte, len(data)-1)
		copy(bitfield, data[1:])
		msg := &BitfieldMsg{
			Bitfield: bitfield,
		}
		return msg, nil

	case 6:
		if len(data) != 13 {
			return nil, fmt.Errorf("Invalid request message length: %d", len(data))
		}
		index := int32(binary.BigEndian.Uint32(data[1:5]))
		offset := int32(binary.BigEndian.Uint32(data[5:9]))
		length := int32(binary.BigEndian.Uint32(data[9:13]))
		msg := &RequestMsg{
			Index:  index,
			Offset: offset,
			Length: length,
		}
		return msg, nil

	case 7:
		if len(data) < 13 {
			return nil, fmt.Errorf("Invalid piece message length: %d", len(data))
		}
		index := int32(binary.BigEndian.Uint32(data[1:5]))
		offset := int32(binary.BigEndian.Uint32(data[5:9]))
		block := make([]byte, len(data)-9)
		copy(block, data[9:])
		msg := &PieceMsg{
			Index:  index,
			Offset: offset,
			Block:  block,
		}
		return msg, nil

	case 8:
		if len(data) != 13 {
			return nil, fmt.Errorf("Invalid cancel message length: %d", len(data))
		}
		index := int32(binary.BigEndian.Uint32(data[1:5]))
		offset := int32(binary.BigEndian.Uint32(data[5:9]))
		length := int32(binary.BigEndian.Uint32(data[9:13]))
		msg := &CancelMsg{
			Index:  index,
			Offset: offset,
			Length: length,
		}
		return msg, nil

	default:
		return nil, fmt.Errorf("Invalid message ID: %d", data[0])
	}
}
