package bencoding

import (
	"bytes"
	"testing"
)

func TestEncodeInt(t *testing.T) {
	inputs := []interface{}{0, 1, -1, 10000000}
	expectedOutputs := []string{"i0e", "i1e", "i-1e", "i10000000e"}

	for i, input := range inputs {
		output, err := Encode(input)
		if err != nil {
			t.Fatalf("Failed to Encode input. Error: %s", err)
		}
		expectedOutput := []byte(expectedOutputs[i])
		if !bytes.Equal(output, expectedOutput) {
			t.Fatalf("Expected %s. Got: %s", expectedOutput, output)
		}
	}
}

func TestEncodeBytes(t *testing.T) {
	inputs := []interface{}{[]byte{}, []byte{1, 2, 3, 4, 5}, []byte("spam")}
	expectedOutputs := []string{"0:", "5:\x01\x02\x03\x04\x05", "4:spam"}

	for i, input := range inputs {
		output, err := Encode(input)
		if err != nil {
			t.Fatalf("Failed to Encode input. Error: %s", err)
		}
		expectedOutput := []byte(expectedOutputs[i])
		if !bytes.Equal(output, expectedOutput) {
			t.Fatalf("Expected %s. Got: %s", expectedOutput, output)
		}
	}
}

func TestEncodeString(t *testing.T) {
	inputs := []interface{}{"", "\x01\x02\x03\x04\x05", "spam"}
	expectedOutputs := []string{"0:", "5:\x01\x02\x03\x04\x05", "4:spam"}

	for i, input := range inputs {
		output, err := Encode(input)
		if err != nil {
			t.Fatalf("Failed to Encode input. Error: %s", err)
		}
		expectedOutput := []byte(expectedOutputs[i])
		if !bytes.Equal(output, expectedOutput) {
			t.Fatalf("Expected %s. Got: %s", expectedOutput, output)
		}
	}
}

func TestEncodeList(t *testing.T) {
	inputs := []interface{}{
		[]interface{}{},
		[]interface{}{[]interface{}{[]interface{}{}}},
		[]interface{}{1, "spam", map[string]interface{}{"foo": "bar"}},
	}
	expectedOutputs := []string{"le", "llleee", "li1e4:spamd3:foo3:baree"}

	for i, input := range inputs {
		output, err := Encode(input)
		if err != nil {
			t.Fatalf("Failed to Encode input. Error: %s", err)
		}
		expectedOutput := []byte(expectedOutputs[i])
		if !bytes.Equal(output, expectedOutput) {
			t.Fatalf("Expected %s. Got: %s", expectedOutput, output)
		}
	}
}

func TestEncodeMap(t *testing.T) {
	inputs := []interface{}{
		map[string]interface{}{},
		map[string]interface{}{"foo": map[string]interface{}{"bar": map[string]interface{}{}}},
		map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": []interface{}{1, 2, map[string]interface{}{"spam": "eggs"}},
			},
		},
	}
	expectedOutputs := []string{"de", "d3:food3:bardeee", "d3:food3:barli1ei2ed4:spam4:eggseeee"}

	for i, input := range inputs {
		output, err := Encode(input)
		if err != nil {
			t.Fatalf("Failed to Encode input. Error: %s", err)
		}
		expectedOutput := []byte(expectedOutputs[i])
		if !bytes.Equal(output, expectedOutput) {
			t.Fatalf("Expected %s. Got: %s", expectedOutput, output)
		}
	}
}

func TestDecodeInt(t *testing.T) {
	expectedOutputs := []int{0, 1, -1, 10000000}
	inputs := []string{"i0e", "i1e", "i-1e", "i10000000e"}

	for i, input := range inputs {
		_output, err := Decode([]byte(input))
		if err != nil {
			t.Fatalf("Failed to Encode input. Error: %s", err)
		}
		output := _output.(int)
		expectedOutput := expectedOutputs[i]
		if expectedOutput != output {
			t.Fatalf("Expected %v. Got: %v", expectedOutput, output)
		}
	}
}

func TestDecodeBytes(t *testing.T) {
	expectedOutputs := []string{"", "\x01\x02\x03\x04\x05", "spam"}
	inputs := []string{"0:", "5:\x01\x02\x03\x04\x05", "4:spam"}

	for i, input := range inputs {
		_output, err := Decode([]byte(input))
		if err != nil {
			t.Fatalf("Failed to Encode input. Error: %s", err)
		}
		output := _output.([]byte)
		expectedOutput := []byte(expectedOutputs[i])
		if !bytes.Equal(output, expectedOutput) {
			t.Fatalf("Expected %v. Got: %v", expectedOutput, output)
		}
	}
}

func TestDecodeList(t *testing.T) {
	expectedOutput := []interface{}{1, 4}

	input := []byte("li1ei4ee")

	_output, err := Decode(input)
	if err != nil {
		t.Fatalf("Failed to Encode input. Error: %s", err)
	}
	output := _output.([]interface{})

	for i, item := range output {
		if expectedOutput[i] != item {
			t.Fatalf("Expected %v. Got: %v", expectedOutput[i], item)
		}
	}
}

func TestDecodeMap(t *testing.T) {
	expectedOutput := map[string]interface{}{"foo": 3}

	input := []byte("d3:fooi3ee")

	_output, err := Decode(input)
	if err != nil {
		t.Fatalf("Failed to Encode input. Error: %s", err)
	}
	output := _output.(map[string]interface{})

	for k, v := range output {
		expectedV := expectedOutput[k]
		if expectedV != v {
			t.Fatalf("Expected %v. Got: %v", expectedV, v)
		}
	}
}
