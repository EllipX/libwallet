package wltutil

import (
	"fmt"
	"math/big"
)

// DecodeEVMEthCallString extracts a single string returned by an EVM eth_call.
func DecodeEVMEthCallString(data []byte) (string, error) {
	if len(data) < 64 { // Minimum size for offset (32 bytes) + length (32 bytes)
		return "", fmt.Errorf("invalid eth_call response: insufficient data for offset and length, got %d bytes", len(data))
	}

	// Extract offset (first 32 bytes) - not used because it's normally 0x20
	// offsetBytes := data[:32]
	// offset := new(big.Int).SetBytes(offsetBytes)

	// Extract string length (second 32 bytes)
	lengthBytes := data[32:64]
	length := new(big.Int).SetBytes(lengthBytes).Uint64()

	if uint64(len(data)) < 64+length {
		return "", fmt.Errorf("invalid eth_call response: data length (%d) less than specified string length (%d) + 64", len(data), length)
	}

	// Extract the string data
	stringData := data[64 : 64+length]
	return string(stringData), nil
}
