package storage_test

import (
	"testing"

	"github.com/nkoji21/leetdaily/internal/storage"
)

func TestEncodeJSON(t *testing.T) {
	t.Parallel()

	data, err := storage.EncodeJSON("test.json", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("EncodeJSON() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("EncodeJSON() returned empty data")
	}
	if data[len(data)-1] != '\n' {
		t.Fatal("EncodeJSON() output does not end with newline")
	}
}

func TestEncodeJSONError(t *testing.T) {
	t.Parallel()

	_, err := storage.EncodeJSON("test.json", make(chan int))
	if err == nil {
		t.Fatal("EncodeJSON() with unencodable value: got nil, want error")
	}
}

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	data := []byte(`{"key":"value"}`)
	var dest map[string]string
	if err := storage.DecodeJSON("test.json", data, &dest); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if dest["key"] != "value" {
		t.Fatalf("DecodeJSON() dest[key] = %q, want %q", dest["key"], "value")
	}
}

func TestDecodeJSONError(t *testing.T) {
	t.Parallel()

	var dest map[string]string
	if err := storage.DecodeJSON("test.json", []byte(`not json`), &dest); err == nil {
		t.Fatal("DecodeJSON() with invalid JSON: got nil, want error")
	}
}

func TestVersionFromBytes(t *testing.T) {
	t.Parallel()

	v1 := storage.VersionFromBytes([]byte("hello"))
	v2 := storage.VersionFromBytes([]byte("hello"))
	v3 := storage.VersionFromBytes([]byte("world"))

	if v1 != v2 {
		t.Fatalf("VersionFromBytes() not deterministic: %v != %v", v1, v2)
	}
	if v1 == v3 {
		t.Fatal("VersionFromBytes() same version for different inputs")
	}
	if v1.Token == "" {
		t.Fatal("VersionFromBytes() returned empty token")
	}
}
