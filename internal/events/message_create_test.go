package events

import (
	"context"
	"strings"
	"testing"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected int
	}{
		{
			name:     "empty text",
			input:    "",
			maxLen:   100,
			expected: 0,
		},
		{
			name:     "short text",
			input:    "Hello, world!",
			maxLen:   100,
			expected: 1,
		},
		{
			name:     "exact limit",
			input:    "12345",
			maxLen:   5,
			expected: 1,
		},
		{
			name:     "split by paragraph",
			input:    "Paragraph 1\n\nParagraph 2\n\nParagraph 3",
			maxLen:   15,
			expected: 3,
		},
		{
			name:     "long text without delimiter",
			input:    strings.Repeat("a", 100),
			maxLen:   30,
			expected: 4, // 30 + 30 + 30 + 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := SplitMessage(tt.input, tt.maxLen)
			if len(chunks) != tt.expected {
				t.Fatalf("expected %d chunks, got %d: %#v", tt.expected, len(chunks), chunks)
			}
			for i, chunk := range chunks {
				if len(chunk) > tt.maxLen {
					t.Errorf("chunk %d exceeds maxLen (%d > %d): %q", i, len(chunk), tt.maxLen, chunk)
				}
			}
		})
	}
}

func TestResolveMentions(t *testing.T) {
	botID := snowflake.ID(123456)
	userID := snowflake.ID(789012)
	userName := "Alice"

	msg := &discord.Message{
		Content: "Hello <@123456>, have you seen <@789012> and <@!789012>?",
		Mentions: []discord.User{
			{
				ID:       userID,
				Username: userName,
			},
		},
	}

	ctx := context.Background()
	res := ResolveMentions(ctx, nil, msg)

	if !strings.Contains(res, "@Alice") {
		t.Errorf("expected @Alice in resolved string, got: %s", res)
	}
	if strings.Contains(res, "<@789012>") || strings.Contains(res, "<@!789012>") {
		t.Errorf("unresolved mentions remained in output: %s", res)
	}
	_ = botID
}

func TestShouldRespond(t *testing.T) {
	// Bot author should be ignored
	botAuthorMsg := &discord.Message{
		Author: discord.User{
			Bot: true,
		},
	}
	if ShouldRespond(nil, botAuthorMsg) {
		t.Errorf("expected false for bot message, got true")
	}

	// Webhook message should be ignored
	webhookID := snowflake.ID(999999)
	webhookMsg := &discord.Message{
		WebhookID: &webhookID,
	}
	if ShouldRespond(nil, webhookMsg) {
		t.Errorf("expected false for webhook message, got true")
	}
}
