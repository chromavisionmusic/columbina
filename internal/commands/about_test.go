package commands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	dgevents "github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
)

func TestAboutCommandMetadata(t *testing.T) {
	if AboutCommandMetadata.CommandName() != "about" {
		t.Errorf("expected command name 'about', got %q", AboutCommandMetadata.CommandName())
	}
	if AboutCommandMetadata.Description != "Information about columbina" {
		t.Errorf("unexpected description: %q", AboutCommandMetadata.Description)
	}
	if len(AboutCommandMetadata.IntegrationTypes) != 2 {
		t.Errorf("expected 2 integration types, got %d", len(AboutCommandMetadata.IntegrationTypes))
	}
	if len(AboutCommandMetadata.Contexts) != 3 {
		t.Errorf("expected 3 contexts, got %d", len(AboutCommandMetadata.Contexts))
	}
}

func TestAboutCommandHandler_Full(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{
			Name:        "Columbina",
			Description: "A gentle singing harbinger",
			Version:     "2.1.0",
			Color:       config.HexColor(0x9c3f73),
			Owner: config.User{
				ID:   "111222333",
				Name: "FatuiOwner",
			},
			Designer: config.User{
				ID:   "444555666",
				Name: "FatuiDesigner",
			},
			Links: config.Links{
				Invite:  "https://discord.gg/columbina-invite",
				Website: "https://columbina.ai",
				Server:  "https://discord.gg/columbina-support",
				Github:  "https://github.com/chromavisionmusic/columbina",
			},
		},
	}

	h := AboutCommandHandler(cfg)
	if h == nil {
		t.Fatal("AboutCommandHandler returned nil")
	}

	gw := &mockGateway{latency: 25 * time.Millisecond}
	c := cache.New()
	client := &bot.Client{
		Gateway: gw,
		Caches:  c,
	}

	var capturedResponse discord.MessageCreate
	genericEvent := dgevents.NewGenericEvent(client, 1, 0)

	var appCmdInteraction discord.ApplicationCommandInteraction
	interactionJSON := `{"id":"123456789012345678","type":2,"token":"tok","data":{"id":"123456789012345678","name":"about","type":1}}`
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
		t.Fatalf("AboutCommandHandler returned error: %v", err)
	}

	// Verify content has Github link
	if !strings.Contains(capturedResponse.Content, "https://github.com/chromavisionmusic/columbina") {
		t.Errorf("message content does not contain github link: %q", capturedResponse.Content)
	}

	// Verify Embed
	if len(capturedResponse.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(capturedResponse.Embeds))
	}
	embed := capturedResponse.Embeds[0]
	if embed.Title != "Columbina" {
		t.Errorf("Title = %q, want 'Columbina'", embed.Title)
	}
	if embed.Description != "A gentle singing harbinger" {
		t.Errorf("Description = %q", embed.Description)
	}
	if embed.Color != 0x9c3f73 {
		t.Errorf("Color = %x, want 0x9c3f73", embed.Color)
	}

	// Verify Embed Fields
	fieldsMap := make(map[string]string)
	for _, f := range embed.Fields {
		fieldsMap[f.Name] = f.Value
	}

	if fieldsMap["Version"] != "2.1.0" {
		t.Errorf("Version field = %q, want '2.1.0'", fieldsMap["Version"])
	}
	if fieldsMap["Ping"] != "25ms" {
		t.Errorf("Ping field = %q, want '25ms'", fieldsMap["Ping"])
	}
	if fieldsMap["Commands"] != "3" {
		t.Errorf("Commands field = %q, want '3'", fieldsMap["Commands"])
	}
	if !strings.Contains(fieldsMap["Developed by"], "FatuiOwner") {
		t.Errorf("Developed by field = %q", fieldsMap["Developed by"])
	}
	if !strings.Contains(fieldsMap["Designed by"], "FatuiDesigner") {
		t.Errorf("Designed by field = %q", fieldsMap["Designed by"])
	}

	// Verify ActionRow buttons
	if len(capturedResponse.Components) != 1 {
		t.Fatalf("expected 1 component action row, got %d", len(capturedResponse.Components))
	}
}

func TestAboutCommandHandler_NoLinks(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{
			Name:    "Columbina",
			Version: "1.0.0",
		},
	}

	h := AboutCommandHandler(cfg)
	client := &bot.Client{
		Caches: cache.New(),
	}

	var capturedResponse discord.MessageCreate
	genericEvent := dgevents.NewGenericEvent(client, 1, 0)

	var appCmdInteraction discord.ApplicationCommandInteraction
	interactionJSON := `{"id":"123456789012345678","type":2,"token":"tok","data":{"id":"123456789012345678","name":"about","type":1}}`
	_ = json.Unmarshal([]byte(interactionJSON), &appCmdInteraction)

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
		t.Fatalf("AboutCommandHandler returned error: %v", err)
	}

	// Should have no action row when all link fields are empty
	if len(capturedResponse.Components) != 0 {
		t.Errorf("expected 0 components when links are empty, got %d", len(capturedResponse.Components))
	}

	// Check ping is 0ms when no gateway
	if len(capturedResponse.Embeds) > 0 {
		for _, f := range capturedResponse.Embeds[0].Fields {
			if f.Name == "Ping" && f.Value != "0ms" {
				t.Errorf("expected Ping 0ms without gateway, got %q", f.Value)
			}
		}
	}
}
