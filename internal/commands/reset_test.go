package commands

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	dgevents "github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
)

func TestResetCommandMetadata(t *testing.T) {
	if ResetCommandMetadata.CommandName() != "reset" {
		t.Errorf("expected command name 'reset', got %q", ResetCommandMetadata.CommandName())
	}
	if ResetCommandMetadata.Description != "Reset your conversation memory with Columbina" {
		t.Errorf("unexpected description: %q", ResetCommandMetadata.Description)
	}
	if len(ResetCommandMetadata.IntegrationTypes) != 2 {
		t.Errorf("expected 2 integration types, got %d", len(ResetCommandMetadata.IntegrationTypes))
	}
	if len(ResetCommandMetadata.Contexts) != 3 {
		t.Errorf("expected 3 contexts, got %d", len(ResetCommandMetadata.Contexts))
	}
}

func TestResetCommandHandler_Execution(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{
			Color: 0x9c3f73,
		},
	}

	h := ResetCommandHandler(cfg, nil)
	if h == nil {
		t.Fatal("ResetCommandHandler returned nil")
	}

	client := &bot.Client{}
	var capturedResponse discord.MessageCreate
	genericEvent := dgevents.NewGenericEvent(client, 1, 0)

	var appCmdInteraction discord.ApplicationCommandInteraction
	interactionJSON := `{"id":"123456789012345678","type":2,"token":"tok","user":{"id":"999888777","username":"wanderer"},"data":{"id":"123456789012345678","name":"reset","type":1}}`
	if err := json.Unmarshal([]byte(interactionJSON), &appCmdInteraction); err != nil {
		t.Fatalf("failed to unmarshal interaction: %v", err)
	}

	interactionCreate := &dgevents.ApplicationCommandInteractionCreate{
		GenericEvent:                  genericEvent,
		ApplicationCommandInteraction: appCmdInteraction,
		Respond: func(responseType discord.InteractionResponseType, data discord.InteractionResponseData, opts ...rest.RequestOpt) error {
			if msg, ok := data.(discord.MessageCreate); ok {
				capturedResponse = msg
			}
			return nil
		},
	}

	event := &handler.CommandEvent{
		ApplicationCommandInteractionCreate: interactionCreate,
		Ctx:                                 context.Background(),
	}

	err := h(discord.SlashCommandInteractionData{}, event)
	if err != nil {
		t.Fatalf("ResetCommandHandler returned error: %v", err)
	}

	if len(capturedResponse.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(capturedResponse.Embeds))
	}

	embed := capturedResponse.Embeds[0]
	if embed.Title != "Memory Reset" {
		t.Errorf("Title = %q, want 'Memory Reset'", embed.Title)
	}
	if embed.Color != 0x9c3f73 {
		t.Errorf("Color = %x, want 0x9c3f73", embed.Color)
	}
	if embed.Description != "My memories of our previous conversation have gently faded into the moonlight. What shall we talk about now?" {
		t.Errorf("Description mismatch: %q", embed.Description)
	}
}
