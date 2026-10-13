package events

import (
	"context"
	"strings"
	"testing"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/snowflake/v2"
)

func newTestClient(botID snowflake.ID) *bot.Client {
	c := cache.New()
	c.SetSelfUser(discord.OAuth2User{
		User: discord.User{
			ID: botID,
		},
	})
	return &bot.Client{
		ApplicationID: botID,
		Caches:        c,
	}
}

func TestShouldRespond(t *testing.T) {
	botID := snowflake.ID(1001)
	client := newTestClient(botID)

	guildID := snowflake.ID(2001)

	t.Run("nil message", func(t *testing.T) {
		if ShouldRespond(client, nil) {
			t.Error("expected false for nil message")
		}
	})

	t.Run("bot author ignored", func(t *testing.T) {
		msg := &discord.Message{
			Author: discord.User{
				ID:  snowflake.ID(9999),
				Bot: true,
			},
		}
		if ShouldRespond(client, msg) {
			t.Error("expected false for bot author")
		}
	})

	t.Run("webhook ignored", func(t *testing.T) {
		webhookID := snowflake.ID(8888)
		msg := &discord.Message{
			WebhookID: &webhookID,
		}
		if ShouldRespond(client, msg) {
			t.Error("expected false for webhook message")
		}
	})

	t.Run("direct message always responds", func(t *testing.T) {
		msg := &discord.Message{
			GuildID: nil, // DM
			Content: "Hello Columbina",
		}
		if !ShouldRespond(client, msg) {
			t.Error("expected true for DM")
		}
	})

	t.Run("nil client in guild returns false", func(t *testing.T) {
		msg := &discord.Message{
			GuildID: &guildID,
			Content: "Hello",
		}
		if ShouldRespond(nil, msg) {
			t.Error("expected false for nil client in guild")
		}
	})

	t.Run("mentioned via mentions array", func(t *testing.T) {
		msg := &discord.Message{
			GuildID: &guildID,
			Mentions: []discord.User{
				{ID: botID},
			},
		}
		if !ShouldRespond(client, msg) {
			t.Error("expected true when bot is in mentions array")
		}
	})

	t.Run("mentioned via text string standard", func(t *testing.T) {
		msg := &discord.Message{
			GuildID: &guildID,
			Content: "Hey <@1001>, how are you?",
		}
		if !ShouldRespond(client, msg) {
			t.Error("expected true when <@botID> is in text")
		}
	})

	t.Run("mentioned via text string nick format", func(t *testing.T) {
		msg := &discord.Message{
			GuildID: &guildID,
			Content: "Hey <@!1001>, how are you?",
		}
		if !ShouldRespond(client, msg) {
			t.Error("expected true when <@!botID> is in text")
		}
	})

	t.Run("reply to bot message inlined", func(t *testing.T) {
		refMsgID := snowflake.ID(5001)
		msg := &discord.Message{
			GuildID: &guildID,
			MessageReference: &discord.MessageReference{
				MessageID: &refMsgID,
			},
			ReferencedMessage: &discord.Message{
				ID: refMsgID,
				Author: discord.User{
					ID: botID,
				},
			},
		}
		if !ShouldRespond(client, msg) {
			t.Error("expected true when replying to bot message")
		}
	})

	t.Run("reply to other user does not respond", func(t *testing.T) {
		refMsgID := snowflake.ID(5002)
		msg := &discord.Message{
			GuildID: &guildID,
			MessageReference: &discord.MessageReference{
				MessageID: &refMsgID,
			},
			ReferencedMessage: &discord.Message{
				ID: refMsgID,
				Author: discord.User{
					ID: snowflake.ID(9999),
				},
			},
		}
		if ShouldRespond(client, msg) {
			t.Error("expected false when replying to someone else")
		}
	})

	t.Run("regular guild message without mention returns false", func(t *testing.T) {
		msg := &discord.Message{
			GuildID: &guildID,
			Content: "Just chatting about tea",
		}
		if ShouldRespond(client, msg) {
			t.Error("expected false for unmentioned message")
		}
	})
}

func TestResolveMentions(t *testing.T) {
	ctx := context.Background()
	botID := snowflake.ID(1001)
	client := newTestClient(botID)

	t.Run("nil message", func(t *testing.T) {
		if got := ResolveMentions(ctx, client, nil); got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("resolve bot mention", func(t *testing.T) {
		msg := &discord.Message{
			Content: "Hello <@1001>!",
		}
		got := ResolveMentions(ctx, client, msg)
		if got != "Hello @Columbina!" {
			t.Errorf("got %q, want 'Hello @Columbina!'", got)
		}
	})

	t.Run("resolve user from Mentions slice", func(t *testing.T) {
		userSF := snowflake.ID(2002)
		msg := &discord.Message{
			Content: "Ask <@2002> about it.",
			Mentions: []discord.User{
				{
					ID:       userSF,
					Username: "Paimon",
				},
			},
		}
		got := ResolveMentions(ctx, client, msg)
		if got != "Ask @Paimon about it." {
			t.Errorf("got %q, want 'Ask @Paimon about it.'", got)
		}
	})

	t.Run("fallback to raw ID when unresolvable", func(t *testing.T) {
		msg := &discord.Message{
			Content: "Unknown <@9876543210> user.",
		}
		got := ResolveMentions(ctx, client, msg)
		if got != "Unknown @9876543210 user." {
			t.Errorf("got %q, want 'Unknown @9876543210 user.'", got)
		}
	})
}

func TestTraverseReplyChain(t *testing.T) {
	ctx := context.Background()
	client := &bot.Client{}

	t.Run("guards for nil parameters", func(t *testing.T) {
		if TraverseReplyChain(ctx, nil, &discord.Message{}, 5) != nil {
			t.Error("expected nil for nil client")
		}
		if TraverseReplyChain(ctx, client, nil, 5) != nil {
			t.Error("expected nil for nil message")
		}
		if TraverseReplyChain(ctx, client, &discord.Message{}, 0) != nil {
			t.Error("expected nil for maxDepth 0")
		}
	})

	t.Run("message without reply reference returns nil", func(t *testing.T) {
		msg := &discord.Message{ID: snowflake.ID(1)}
		if chain := TraverseReplyChain(ctx, client, msg, 5); chain != nil {
			t.Errorf("expected nil chain, got %v", chain)
		}
	})

	t.Run("traverses inlined references in chronological order", func(t *testing.T) {
		id1 := snowflake.ID(10)
		id2 := snowflake.ID(20)
		id3 := snowflake.ID(30)

		m1 := &discord.Message{
			ID:      id1,
			Content: "first message",
			Author:  discord.User{Username: "User1"},
		}

		m2 := &discord.Message{
			ID:                id2,
			Content:           "second message",
			Author:            discord.User{Username: "User2"},
			MessageReference:  &discord.MessageReference{MessageID: &id1},
			ReferencedMessage: m1,
		}

		m3 := &discord.Message{
			ID:                id3,
			Content:           "third message",
			Author:            discord.User{Username: "User3"},
			MessageReference:  &discord.MessageReference{MessageID: &id2},
			ReferencedMessage: m2,
		}

		chain := TraverseReplyChain(ctx, client, m3, 10)
		if len(chain) != 2 {
			t.Fatalf("expected 2 parent messages, got %d", len(chain))
		}

		// Should be chronological: m1, then m2
		if chain[0].ID != id1 {
			t.Errorf("chain[0].ID = %v, want %v", chain[0].ID, id1)
		}
		if chain[1].ID != id2 {
			t.Errorf("chain[1].ID = %v, want %v", chain[1].ID, id2)
		}
	})

	t.Run("guards against circular reference loops", func(t *testing.T) {
		idA := snowflake.ID(100)
		idB := snowflake.ID(200)

		var msgA, msgB discord.Message
		msgA = discord.Message{
			ID:                idA,
			MessageReference:  &discord.MessageReference{MessageID: &idB},
			ReferencedMessage: &msgB,
		}
		msgB = discord.Message{
			ID:                idB,
			MessageReference:  &discord.MessageReference{MessageID: &idA},
			ReferencedMessage: &msgA,
		}

		chain := TraverseReplyChain(ctx, client, &msgA, 10)
		if len(chain) != 1 {
			t.Fatalf("expected loop detection to stop at len 1, got %d", len(chain))
		}
	})
}

func TestBuildContext(t *testing.T) {
	ctx := context.Background()
	botID := snowflake.ID(9000)
	client := newTestClient(botID)

	t.Run("without reply chain", func(t *testing.T) {
		msg := &discord.Message{
			Content: "Hello there",
		}
		got := BuildContext(ctx, client, msg)
		if got != "Hello there" {
			t.Errorf("got %q, want 'Hello there'", got)
		}
	})

	t.Run("with reply chain", func(t *testing.T) {
		parentID := snowflake.ID(500)
		parentMsg := &discord.Message{
			ID:      parentID,
			Content: "What is your name?",
			Author: discord.User{
				ID:       snowflake.ID(123),
				Username: "Traveler",
			},
		}

		currentMsg := &discord.Message{
			ID:      snowflake.ID(600),
			Content: "I am asking you",
			Author: discord.User{
				ID:       snowflake.ID(123),
				Username: "Traveler",
			},
			MessageReference:  &discord.MessageReference{MessageID: &parentID},
			ReferencedMessage: parentMsg,
		}

		got := BuildContext(ctx, client, currentMsg)
		if !strings.Contains(got, "[Prior conversation history in Discord reply chain]") {
			t.Errorf("expected prior history header in: %s", got)
		}
		if !strings.Contains(got, "[Traveler]: What is your name?") {
			t.Errorf("expected parent message in: %s", got)
		}
		if !strings.Contains(got, "[Current message]\n[Traveler]: I am asking you") {
			t.Errorf("expected current message in: %s", got)
		}
	})
}

func TestSplitMessage(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		if got := SplitMessage("", 100); got != nil {
			t.Errorf("expected nil for empty string, got %v", got)
		}
		if got := SplitMessage("   \n\t  ", 100); got != nil {
			t.Errorf("expected nil for whitespace string, got %v", got)
		}
	})

	t.Run("message under maxLen returns single chunk", func(t *testing.T) {
		msg := "Short message"
		chunks := SplitMessage(msg, 100)
		if len(chunks) != 1 || chunks[0] != msg {
			t.Errorf("got %v, want [%q]", chunks, msg)
		}
	})

	t.Run("splits on double newlines", func(t *testing.T) {
		part1 := strings.Repeat("A", 40)
		part2 := strings.Repeat("B", 40)
		input := part1 + "\n\n" + part2

		chunks := SplitMessage(input, 50)
		if len(chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
		}
		if chunks[0] != part1 || chunks[1] != part2 {
			t.Errorf("chunks mismatch: %v", chunks)
		}
	})

	t.Run("splits on single newlines", func(t *testing.T) {
		part1 := strings.Repeat("A", 40)
		part2 := strings.Repeat("B", 40)
		input := part1 + "\n" + part2

		chunks := SplitMessage(input, 50)
		if len(chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
		}
		if chunks[0] != part1 || chunks[1] != part2 {
			t.Errorf("chunks mismatch: %v", chunks)
		}
	})

	t.Run("splits on spaces", func(t *testing.T) {
		part1 := strings.Repeat("A", 40)
		part2 := strings.Repeat("B", 40)
		input := part1 + " " + part2

		chunks := SplitMessage(input, 50)
		if len(chunks) != 2 {
			t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
		}
		if chunks[0] != part1 || chunks[1] != part2 {
			t.Errorf("chunks mismatch: %v", chunks)
		}
	})

	t.Run("splits hard boundary when no delimiter exists", func(t *testing.T) {
		input := strings.Repeat("X", 80)
		chunks := SplitMessage(input, 30)
		if len(chunks) != 3 {
			t.Fatalf("expected 3 chunks, got %d", len(chunks))
		}
		if len(chunks[0]) != 30 || len(chunks[1]) != 30 || len(chunks[2]) != 20 {
			t.Errorf("chunk lengths = [%d, %d, %d]", len(chunks[0]), len(chunks[1]), len(chunks[2]))
		}
	})

	t.Run("default maxLen when non-positive", func(t *testing.T) {
		input := strings.Repeat("Y", 2500)
		chunks := SplitMessage(input, 0)
		if len(chunks) < 2 {
			t.Fatalf("expected message to be split with default 1950, got %d chunks", len(chunks))
		}
	})
}
