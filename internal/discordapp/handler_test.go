package discordapp

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/storage"
)

func TestHandlerRespondsToPing(t *testing.T) {
	t.Parallel()

	handler, privateKey := newTestHandler(t)
	recorder := httptest.NewRecorder()
	request := signedRequest(t, privateKey, `{"type":1}`)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"type":1`) {
		t.Fatalf("response = %s, want pong response", recorder.Body.String())
	}
}

func TestHandlerSavesSetup(t *testing.T) {
	t.Parallel()

	handler, privateKey := newTestHandler(t)
	recorder := httptest.NewRecorder()
	request := signedRequest(t, privateKey, `{
		"type": 2,
		"guild_id": "123456789012345678",
		"member": {"permissions": "32"},
		"data": {
			"name": "setup",
			"options": [
				{"name": "forum", "value": "234567890123456789"},
				{"name": "notify", "value": "345678901234567890"},
				{"name": "start", "value": 42}
			],
			"resolved": {
				"channels": {
					"234567890123456789": {"guild_id": "123456789012345678", "type": 15},
					"345678901234567890": {"guild_id": "123456789012345678", "type": 0}
				}
			}
		}
	}`)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Type int `json:"type"`
		Data struct {
			Content string `json:"content"`
			Flags   *int   `json:"flags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.Data.Flags != nil {
		t.Fatalf("response flags = %d, want nil", *response.Data.Flags)
	}
	if !strings.Contains(response.Data.Content, "setup saved:") {
		t.Fatalf("response content = %q, want setup confirmation", response.Data.Content)
	}

	repository := handler.repo.(*stubRepository)
	if len(repository.guilds.Guilds) != 1 {
		t.Fatalf("len(guilds) = %d, want 1", len(repository.guilds.Guilds))
	}
	if repository.guilds.Guilds[0].StartProblemNumber != 42 {
		t.Fatalf("start problem = %d, want 42", repository.guilds.Guilds[0].StartProblemNumber)
	}
}

func TestHandlerRejectsMissingPermission(t *testing.T) {
	t.Parallel()

	handler, privateKey := newTestHandler(t)
	recorder := httptest.NewRecorder()
	request := signedRequest(t, privateKey, `{
		"type": 2,
		"guild_id": "123456789012345678",
		"member": {"permissions": "0"},
		"data": {"name": "setup"}
	}`)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Data struct {
			Content string `json:"content"`
			Flags   int    `json:"flags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.Data.Flags != messageFlagEphemeral {
		t.Fatalf("response flags = %d, want %d", response.Data.Flags, messageFlagEphemeral)
	}
	if !strings.Contains(response.Data.Content, "Manage Server") {
		t.Fatalf("response = %s, want permission error", recorder.Body.String())
	}
}

func TestHandlerRejectsInvalidSignature(t *testing.T) {
	t.Parallel()

	handler, _ := newTestHandler(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/discord/interactions", bytes.NewBufferString(`{"type":1}`))
	request.Header.Set("X-Signature-Ed25519", "00")
	request.Header.Set("X-Signature-Timestamp", "1")

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func newTestHandler(t *testing.T) (*Handler, ed25519.PrivateKey) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	handler, err := NewHandler(hex.EncodeToString(publicKey), &stubRepository{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	return handler, privateKey
}

func signedRequest(t *testing.T, privateKey ed25519.PrivateKey, body string) *http.Request {
	t.Helper()

	timestamp := "1700000000"
	signature := ed25519.Sign(privateKey, []byte(timestamp+body))
	request := httptest.NewRequest(http.MethodPost, "/discord/interactions", bytes.NewBufferString(body))
	request.Header.Set("X-Signature-Ed25519", hex.EncodeToString(signature))
	request.Header.Set("X-Signature-Timestamp", timestamp)
	return request
}

type stubRepository struct {
	guilds       config.GuildSettings
	guildVersion storage.Version
}

func (s *stubRepository) LoadGuildSettings(context.Context) (config.GuildSettings, storage.Version, error) {
	return s.guilds, s.guildVersion, nil
}

func (s *stubRepository) SaveGuildSettings(_ context.Context, guilds config.GuildSettings, version storage.Version) (storage.Version, error) {
	s.guilds = guilds
	s.guildVersion = storage.Version{Token: version.Token + "next"}
	return s.guildVersion, nil
}
