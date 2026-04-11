package discord

import (
	"context"
	"fmt"

	"github.com/nkoji21/leetdaily/internal/discordid"
)

type Notifier struct {
	client    *Client
	channelID string
}

func NewNotifier(client *Client, channelID string) (*Notifier, error) {
	if client == nil {
		return nil, fmt.Errorf("Discord notifier client must not be nil")
	}
	if !discordid.IsValid(channelID) {
		return nil, fmt.Errorf("Discord notification channel ID %q is not a valid snowflake", channelID)
	}

	return &Notifier{client: client, channelID: channelID}, nil
}

func (n *Notifier) NotifyFailure(ctx context.Context, guildID string, err error) error {
	if err == nil {
		return fmt.Errorf("failure notification requires a non-nil error")
	}

	content := fmt.Sprintf("LeetDaily failed for guild `%s`: %v", guildID, err)
	return n.client.SendMessage(ctx, n.channelID, content)
}
