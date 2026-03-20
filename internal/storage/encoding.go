package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func EncodeJSON(path string, value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %q: %w", path, err)
	}

	return append(data, '\n'), nil
}

func DecodeJSON(path string, data []byte, destination any) error {
	if err := json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("decode %q: %w", path, err)
	}

	return nil
}

func VersionFromBytes(data []byte) Version {
	sum := sha256.Sum256(data)
	return Version{Token: hex.EncodeToString(sum[:])}
}
