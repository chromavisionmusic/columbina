package events

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	dgevents "github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

var userMentionRegex = regexp.MustCompile(`<@!?(\d+)>`)

// ShouldRespond checks if the incoming message is a reply to the bot or mentions the bot.
func ShouldRespond(client *bot.Client, msg *discord.Message) bool {
	if client == nil || msg == nil {
		return false
	}

	// Ignore bots and webhooks
	if msg.Author.Bot || msg.WebhookID != nil {
		return false
	}

	botID := client.ID()

	// Check if bot is directly in mentions array
	for _, user := range msg.Mentions {
		if user.ID == botID {
			return true
		}
	}

	// Check if bot is mentioned via string pattern
	botMention := fmt.Sprintf("<@%s>", botID)
	botNickMention := fmt.Sprintf("<@!%s>", botID)
	if strings.Contains(msg.Content, botMention) || strings.Contains(msg.Content, botNickMention) {
		return true
	}

	// Check if this message is a reply to the bot
	if msg.MessageReference != nil && msg.MessageReference.MessageID != nil {
		if msg.ReferencedMessage != nil && msg.ReferencedMessage.Author.ID == botID {
			return true
		}
		// If ReferencedMessage wasn't inlined, check via REST
		if msg.ReferencedMessage == nil {
			if ref, err := client.Rest.GetMessage(msg.ChannelID, *msg.MessageReference.MessageID); err == nil && ref != nil {
				if ref.Author.ID == botID {
					return true
				}
			}
		}
	}

	return false
}

// ResolveMentions replaces <@user_id> and <@!user_id> with the corresponding @username.
func ResolveMentions(ctx context.Context, client *bot.Client, msg *discord.Message) string {
	if msg == nil {
		return ""
	}

	content := msg.Content
	var botID snowflake.ID
	if client != nil {
		botID = client.ID()
	}

	return userMentionRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := userMentionRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		idStr := submatches[1]
		sfID, err := snowflake.Parse(idStr)
		if err != nil {
			return match
		}

		// Bot itself
		if botID != 0 && sfID == botID {
			return "@Columbina"
		}

		// Check message mentions array first
		for _, u := range msg.Mentions {
			if u.ID == sfID {
				return "@" + u.EffectiveName()
			}
		}

		// Check member cache if in a guild
		if client != nil && client.Caches != nil && msg.GuildID != nil {
			if m, ok := client.Caches.Member(*msg.GuildID, sfID); ok {
				return "@" + m.EffectiveName()
			}
		}

		// Fetch via REST as fallback
		if client != nil {
			if u, err := client.Rest.GetUser(sfID); err == nil && u != nil {
				return "@" + u.EffectiveName()
			}
		}

		return "@" + idStr
	})
}

// TraverseReplyChain fetches up to maxDepth parent messages along the reply chain in chronological order.
func TraverseReplyChain(ctx context.Context, client *bot.Client, msg *discord.Message, maxDepth int) []*discord.Message {
	if client == nil || msg == nil || maxDepth <= 0 {
		return nil
	}

	var chain []*discord.Message
	visited := make(map[snowflake.ID]bool)
	visited[msg.ID] = true

	current := msg
	for len(chain) < maxDepth {
		if current.MessageReference == nil || current.MessageReference.MessageID == nil {
			break
		}

		parentID := *current.MessageReference.MessageID
		if visited[parentID] {
			break // Guard against circular loops
		}
		visited[parentID] = true

		var parentMsg *discord.Message
		if current.ReferencedMessage != nil && current.ReferencedMessage.ID == parentID {
			parentMsg = current.ReferencedMessage
		} else {
			fetched, err := client.Rest.GetMessage(current.ChannelID, parentID)
			if err != nil || fetched == nil {
				break
			}
			parentMsg = fetched
		}

		chain = append(chain, parentMsg)
		current = parentMsg
	}

	// Reverse chain to ensure chronological order (oldest to newest)
	slices.Reverse(chain)
	return chain
}

// BuildContext constructs the dialogue context incorporating traversed reply chain messages if present.
func BuildContext(ctx context.Context, client *bot.Client, msg *discord.Message) string {
	chain := TraverseReplyChain(ctx, client, msg, 10)
	currentResolved := ResolveMentions(ctx, client, msg)

	if len(chain) == 0 {
		return currentResolved
	}

	var sb strings.Builder
	sb.WriteString("[Prior conversation history in Discord reply chain]\n")
	for _, prev := range chain {
		authorName := prev.Author.EffectiveName()
		if client != nil && prev.Author.ID == client.ID() {
			authorName = "Columbina"
		}
		resolved := ResolveMentions(ctx, client, prev)
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", authorName, resolved))
	}

	sb.WriteString("\n[Current message]\n")
	authorName := msg.Author.EffectiveName()
	sb.WriteString(fmt.Sprintf("[%s]: %s", authorName, currentResolved))

	return sb.String()
}

// SplitMessage breaks a long response string into chunks of at most maxLen characters,
// respecting paragraph breaks, newlines, and spaces where possible.
func SplitMessage(text string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = 1950
	}

	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		if text == "" {
			return nil
		}
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > maxLen {
		cut := strings.LastIndex(remaining[:maxLen], "\n\n")
		if cut == -1 || cut < maxLen/2 {
			cut = strings.LastIndex(remaining[:maxLen], "\n")
		}
		if cut == -1 || cut < maxLen/2 {
			cut = strings.LastIndex(remaining[:maxLen], " ")
		}
		if cut == -1 || cut < maxLen/2 {
			cut = maxLen
		}

		chunk := strings.TrimSpace(remaining[:cut])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		remaining = strings.TrimSpace(remaining[cut:])
	}

	if remaining != "" {
		chunks = append(chunks, remaining)
	}

	return chunks
}

// OnMessageCreate handles incoming Discord messages, replying when mentioned or replied to.
func (h *Handler) OnMessageCreate(e *dgevents.MessageCreate) {
	if h.aiService == nil {
		return
	}

	if !ShouldRespond(e.Client(), &e.Message) {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Send initial typing indicator
		_ = e.Client().Rest.SendTyping(e.ChannelID)

		// Keep sending typing indicator every 4 seconds while generating
		ticker := time.NewTicker(4 * time.Second)
		done := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					_ = e.Client().Rest.SendTyping(e.ChannelID)
				case <-done:
					ticker.Stop()
					return
				}
			}
		}()
		defer close(done)

		// Build prompt context from reply chain and resolved mentions
		prompt := BuildContext(ctx, e.Client(), &e.Message)

		// Execute ADK runner under the user's isolated session
		reply, err := h.aiService.Ask(ctx, e.Message.Author.ID.String(), prompt)
		if err != nil {
			log.Errorf("Failed to generate AI response for user %s: %v", e.Message.Author.ID, err)
			reply = "I'm sorry, I wandered into the shadows for a moment and couldn't process that. Please try again."
		}

		// Split into Discord-compliant chunks (<= 2000 chars)
		chunks := SplitMessage(reply, 1950)
		if len(chunks) == 0 {
			return
		}

		// Send first chunk as an inline reply to the triggering message
		firstMsg := discord.NewMessageCreate().
			WithContent(chunks[0]).
			WithMessageReferenceByID(e.MessageID)

		_, err = e.Client().Rest.CreateMessage(e.ChannelID, firstMsg)
		if err != nil {
			log.Errorf("Failed to send reply to %s: %v", e.ChannelID, err)
			return
		}

		// Send any subsequent chunks sequentially
		for _, chunk := range chunks[1:] {
			followUp := discord.NewMessageCreate().WithContent(chunk)
			if _, err := e.Client().Rest.CreateMessage(e.ChannelID, followUp); err != nil {
				log.Errorf("Failed to send follow-up message chunk: %v", err)
				break
			}
		}
	}()
}
