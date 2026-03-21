package discordapp

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/storage"
)

const (
	interactionTypePing               = 1
	interactionTypeApplicationCommand = 2

	responseTypePong                     = 1
	responseTypeChannelMessageWithSource = 4

	messageFlagEphemeral = 1 << 6

	permissionAdministrator = 1 << 3
	permissionManageGuild   = 1 << 5

	channelTypeGuildText         = 0
	channelTypeGuildAnnouncement = 5
	channelTypeGuildForum        = 15
)

type GuildSettingsRepository interface {
	LoadGuildSettings(context.Context) (config.GuildSettings, storage.Version, error)
	SaveGuildSettings(context.Context, config.GuildSettings, storage.Version) (storage.Version, error)
}

type Handler struct {
	publicKey ed25519.PublicKey
	repo      GuildSettingsRepository
}

type interaction struct {
	Type    int    `json:"type"`
	GuildID string `json:"guild_id"`
	Member  struct {
		Permissions string `json:"permissions"`
	} `json:"member"`
	Data struct {
		Name     string              `json:"name"`
		Options  []commandOption     `json:"options"`
		Resolved resolvedInteraction `json:"resolved"`
	} `json:"data"`
}

type resolvedInteraction struct {
	Channels map[string]resolvedChannel `json:"channels"`
}

type resolvedChannel struct {
	GuildID string `json:"guild_id"`
	Type    int    `json:"type"`
}

type commandOption struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

type interactionResponse struct {
	Type int                     `json:"type"`
	Data *interactionMessageData `json:"data,omitempty"`
}

type interactionMessageData struct {
	Content string `json:"content"`
	Flags   int    `json:"flags,omitempty"`
}

func NewHandler(publicKey string, repo GuildSettingsRepository) (*Handler, error) {
	if repo == nil {
		return nil, fmt.Errorf("Discord setup repository must not be nil")
	}

	key, err := parsePublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	return &Handler{publicKey: key, repo: repo}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	body, err := readAndVerifyBody(w, r, h.publicKey)
	if err != nil {
		return
	}

	var request interaction
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "invalid Discord interaction payload", http.StatusBadRequest)
		return
	}

	response := interactionResponse{}
	switch request.Type {
	case interactionTypePing:
		response.Type = responseTypePong
	case interactionTypeApplicationCommand:
		response = h.handleCommand(r.Context(), request)
	default:
		response = ephemeralMessage("unsupported interaction type")
	}

	writeJSON(w, response)
}

func (h *Handler) handleCommand(ctx context.Context, request interaction) interactionResponse {
	switch strings.ToLower(strings.TrimSpace(request.Data.Name)) {
	case "setup":
		return h.handleSetup(ctx, request)
	default:
		return ephemeralMessage("unsupported command")
	}
}

func (h *Handler) handleSetup(ctx context.Context, request interaction) interactionResponse {
	if strings.TrimSpace(request.GuildID) == "" {
		return ephemeralMessage("setup must be run from a server")
	}

	if !hasManageGuildPermission(request.Member.Permissions) {
		return ephemeralMessage("setup requires Manage Server permission")
	}

	options, err := parseSetupOptions(request)
	if err != nil {
		return ephemeralMessage(err.Error())
	}

	guild := config.Guild{
		GuildID:               request.GuildID,
		Enabled:               true,
		ForumChannelID:        options.ForumChannelID,
		NotificationChannelID: options.NotificationChannelID,
		StartProblemNumber:    options.StartProblemNumber,
	}

	if err := saveGuildSettings(ctx, h.repo, guild); err != nil {
		return ephemeralMessage(fmt.Sprintf("failed to save setup: %v", err))
	}

	return ephemeralMessage(fmt.Sprintf(
		"setup saved: forum <#%s>, notifications <#%s>, start problem %d",
		guild.ForumChannelID,
		guild.NotificationChannelID,
		guild.StartProblemNumber,
	))
}

type setupOptions struct {
	ForumChannelID        string
	NotificationChannelID string
	StartProblemNumber    int
}

func parseSetupOptions(request interaction) (setupOptions, error) {
	values := map[string]json.RawMessage{}
	for _, option := range request.Data.Options {
		values[option.Name] = option.Value
	}

	forumChannelID, err := parseStringOption(values["forum"])
	if err != nil {
		return setupOptions{}, fmt.Errorf("forum channel is required")
	}
	notificationChannelID, err := parseStringOption(values["notify"])
	if err != nil {
		return setupOptions{}, fmt.Errorf("notification channel is required")
	}
	startProblemNumber, err := parseIntOption(values["start"])
	if err != nil {
		return setupOptions{}, fmt.Errorf("start problem number must be a positive integer")
	}
	if startProblemNumber < 1 {
		return setupOptions{}, fmt.Errorf("start problem number must be greater than 0")
	}

	resolved := request.Data.Resolved.Channels
	forumChannel, ok := resolved[forumChannelID]
	if !ok {
		return setupOptions{}, fmt.Errorf("forum channel could not be resolved")
	}
	if forumChannel.GuildID != request.GuildID {
		return setupOptions{}, fmt.Errorf("forum channel must belong to this server")
	}
	if forumChannel.Type != channelTypeGuildForum {
		return setupOptions{}, fmt.Errorf("forum channel must be a forum channel")
	}

	notificationChannel, ok := resolved[notificationChannelID]
	if !ok {
		return setupOptions{}, fmt.Errorf("notification channel could not be resolved")
	}
	if notificationChannel.GuildID != request.GuildID {
		return setupOptions{}, fmt.Errorf("notification channel must belong to this server")
	}
	if notificationChannel.Type != channelTypeGuildText && notificationChannel.Type != channelTypeGuildAnnouncement {
		return setupOptions{}, fmt.Errorf("notification channel must be a text channel")
	}

	return setupOptions{
		ForumChannelID:        forumChannelID,
		NotificationChannelID: notificationChannelID,
		StartProblemNumber:    startProblemNumber,
	}, nil
}

func parseStringOption(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("missing option")
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}

	return strings.TrimSpace(value), nil
}

func parseIntOption(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, errors.New("missing option")
	}

	var value int
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, nil
	}

	var asFloat float64
	if err := json.Unmarshal(raw, &asFloat); err == nil {
		return int(asFloat), nil
	}

	return 0, errors.New("invalid option")
}

func saveGuildSettings(ctx context.Context, repo GuildSettingsRepository, guild config.Guild) error {
	for attempt := 0; attempt < 3; attempt++ {
		settings, version, err := repo.LoadGuildSettings(ctx)
		if err != nil {
			if !storage.IsNotFound(err) {
				return err
			}
			settings = config.GuildSettings{}
			version = storage.Version{}
		}

		settings.Upsert(guild)
		if _, err := repo.SaveGuildSettings(ctx, settings, version); err != nil {
			if storage.IsConflict(err) {
				continue
			}
			return err
		}
		return nil
	}

	return fmt.Errorf("guild settings update conflicted repeatedly")
}

func hasManageGuildPermission(raw string) bool {
	permissions, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return false
	}

	return permissions&permissionAdministrator != 0 || permissions&permissionManageGuild != 0
}

func parsePublicKey(raw string) (ed25519.PublicKey, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("decode DISCORD_APPLICATION_PUBLIC_KEY: %w", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("DISCORD_APPLICATION_PUBLIC_KEY must decode to %d bytes", ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(decoded), nil
}

func readAndVerifyBody(w http.ResponseWriter, r *http.Request, publicKey ed25519.PublicKey) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return nil, err
	}

	signature, err := hex.DecodeString(strings.TrimSpace(r.Header.Get("X-Signature-Ed25519")))
	if err != nil || len(signature) != ed25519.SignatureSize {
		http.Error(w, "invalid Discord signature", http.StatusUnauthorized)
		return nil, errors.New("invalid signature header")
	}
	timestamp := r.Header.Get("X-Signature-Timestamp")
	if strings.TrimSpace(timestamp) == "" {
		http.Error(w, "missing Discord timestamp", http.StatusUnauthorized)
		return nil, errors.New("missing timestamp header")
	}

	message := append([]byte(timestamp), body...)
	if !ed25519.Verify(publicKey, message, signature) {
		http.Error(w, "invalid Discord signature", http.StatusUnauthorized)
		return nil, errors.New("signature verification failed")
	}

	return body, nil
}
func ephemeralMessage(content string) interactionResponse {
	return interactionResponse{
		Type: responseTypeChannelMessageWithSource,
		Data: &interactionMessageData{
			Content: content,
			Flags:   messageFlagEphemeral,
		},
	}
}

func writeJSON(w http.ResponseWriter, response interactionResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
