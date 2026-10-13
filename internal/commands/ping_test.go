package commands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	dgevents "github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
)

type mockGateway struct {
	gateway.Gateway
	latency time.Duration
}

func (m *mockGateway) Latency() time.Duration {
	return m.latency
}

func TestPingCommandMetadata(t *testing.T) {
	if PingCommandMetadata.CommandName() != "ping" {
		t.Errorf("expected command name 'ping', got %q", PingCommandMetadata.CommandName())
	}
	if PingCommandMetadata.Description != "Check bot latency" {
		t.Errorf("unexpected description: %q", PingCommandMetadata.Description)
	}
	if len(PingCommandMetadata.IntegrationTypes) != 2 {
		t.Errorf("expected 2 integration types, got %d", len(PingCommandMetadata.IntegrationTypes))
	}
	if len(PingCommandMetadata.Contexts) != 3 {
		t.Errorf("expected 3 contexts, got %d", len(PingCommandMetadata.Contexts))
	}
}

func TestPingCommandHandler_Execution(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{
			Color: 0x123456,
		},
	}

	h := PingCommandHandler(cfg)
	if h == nil {
		t.Fatal("PingCommandHandler returned nil")
	}

	gw := &mockGateway{latency: 42 * time.Millisecond}
	client := &bot.Client{
		Gateway: gw,
	}

	var capturedResponse discord.MessageCreate
	genericEvent := dgevents.NewGenericEvent(client, 1, 0)

	var appCmdInteraction discord.ApplicationCommandInteraction
	interactionJSON := `{"id":"123456789012345678","type":2,"token":"tok","data":{"id":"123456789012345678","name":"ping","type":1}}`
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
		t.Fatalf("PingCommandHandler returned error: %v", err)
	}

	if len(capturedResponse.Embeds) != 1 {
		t.Fatalf("expected 1 embed in response, got %d", len(capturedResponse.Embeds))
	}

	embed := capturedResponse.Embeds[0]
	if embed.Title != "Pong!" {
		t.Errorf("expected embed title 'Pong!', got %q", embed.Title)
	}
	if embed.Color != 0x123456 {
		t.Errorf("expected color 0x123456, got %x", embed.Color)
	}
	if len(embed.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(embed.Fields))
	}
	if embed.Fields[0].Name != "API Latency" || embed.Fields[0].Value != "`42ms`" {
		t.Errorf("field 0 mismatch: %+v", embed.Fields[0])
	}
	if embed.Fields[1].Name != "Bot Latency" {
		t.Errorf("field 1 name = %q, want 'Bot Latency'", embed.Fields[1].Name)
	}
}
