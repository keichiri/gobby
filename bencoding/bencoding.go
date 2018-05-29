package bencoding

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"unicode"
)

func Bencode(item interface{}) ([]byte, error) {
	switch t := item.(type) {
	case int:
		return bencodeInt(t), nil
	case string:
		return bencodeBytes([]byte(t)), nil
	case []byte:
		return bencodeBytes(t), nil
	case []interface{}:
		return bencodeList(t)
	case map[string]interface{}:
		return bencodeMap(t)
	default:
		return nil, fmt.Errorf("Unsupported type: %v", reflect.TypeOf(item))
	}
}

func bencodeInt(i int) []byte {
	return []byte(fmt.Sprintf("i%de", i))
}

func bencodeBytes(b []byte) []byte {
	return append([]byte(strconv.Itoa(len(b))+":"), b...)
}

func bencodeList(l []interface{}) ([]byte, error) {
	bencodedChunks := make([][]byte, 0, len(l)+2)
	bencodedChunks = append(bencodedChunks, []byte("l"))

	for _, item := range l {
		bencodedItem, err := Bencode(item)
		if err != nil {
			return nil, err
		}
		bencodedChunks = append(bencodedChunks, bencodedItem)
	}

	bencodedChunks = append(bencodedChunks, []byte("e"))
	return bytes.Join(bencodedChunks, nil), nil
}

func bencodeMap(m map[string]interface{}) ([]byte, error) {
	bencodedChunks := make([][]byte, 0, len(m)*2+2)
	bencodedChunks = append(bencodedChunks, []byte("d"))

	for k, v := range m {
		bencodedK := bencodeBytes([]byte(k))
		bencodedV, err := Bencode(v)
		if err != nil {
			return nil, err
		}

		bencodedChunks = append(bencodedChunks, bencodedK, bencodedV)
	}

	bencodedChunks = append(bencodedChunks, []byte("e"))
	return bytes.Join(bencodedChunks, nil), nil
}

func Bdecode(data []byte) (interface{}, error) {
	item, leftover, err := decode(data)
	if err != nil {
		return nil, err
	}

	if len(leftover) > 0 {
		return nil, fmt.Errorf("Partial decode. %d bytes left", len(leftover))
	}

	return item, nil
}

func decode(data []byte) (interface{}, []byte, error) {
	firstByte := data[0]
	switch {
	case firstByte == 'i':
		return decodeInt(data[1:])
	case firstByte >= '0' && firstByte <= '9':
		return decodeBytes(data)
	case firstByte == 'l':
		return decodeList(data[1:])
	case firstByte == 'd':
		return decodeMap(data[1:])
	default:
		return nil, nil, fmt.Errorf("Invalid item start: %d. Left: %d", firstByte, len(data))
	}
}

func decodeInt(data []byte) (int, []byte, error) {
	split := bytes.SplitN(data, []byte{'e'}, 2)
	if len(split) != 2 {
		return 0, nil, fmt.Errorf("Failed to find integer end. Left: %d", len(data))
	}

	result, err := strconv.Atoi(string(split[0]))
	if err != nil {
		return 0, nil, fmt.Errorf("Invalid integer. Left: %d. Error: %s", len(data), err)
	}

	return result, split[1], nil
}

func decodeBytes(data []byte) (interface{}, []byte, error) {
	split := bytes.SplitN(data, []byte(":"), 2)
	if len(split) != 2 {
		return nil, nil, fmt.Errorf("Failed to find string length end. Left: %d", len(data))
	}

	length, err := strconv.Atoi(string(split[0]))
	if err != nil {
		return nil, nil, fmt.Errorf("Invalid string length. Left: %d. Error: %s", len(data), err)
	}

	if len(split[1]) < length {
		return nil, nil, fmt.Errorf("String too short. Left: %d. Expected: %d", len(split[1]), length)
	}

	result := split[1][:length]
	isAscii := true
	for _, byteValue := range result {
		if byteValue > unicode.MaxASCII {
			isAscii = false
		}
	}
	if isAscii {
		return string(result), split[1][length:], nil
	} else {
		return result, split[1][length:], nil
	}
}

func decodeList(data []byte) ([]interface{}, []byte, error) {
	items := make([]interface{}, 0)
	var item interface{}
	var err error

	for data[0] != 'e' {
		item, data, err = decode(data)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}

	return items, data[1:], nil
}

func decodeMap(data []byte) (map[string]interface{}, []byte, error) {
	m := make(map[string]interface{})
	var _key interface{}
	var value interface{}
	var err error

	for data[0] != 'e' {
		_key, data, err = decodeBytes(data)
		if err != nil {
			return nil, nil, err
		}
		switch _key.(type) {
		case []byte:
			return nil, nil, fmt.Errorf("Invalid map key: %v. Left: %d", _key, len(data))
		case string:
		}

		value, data, err = decode(data)
		if err != nil {
			return nil, nil, err
		}
		key := _key.(string)
		m[key] = value
	}

	return m, data[1:], nil
}
