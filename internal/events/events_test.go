package events

import (
	"context"
	"testing"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/bot"
	dgevents "github.com/disgoorg/disgo/events"
)

func TestNewHandler_WithoutWebhook(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{Name: "Columbina"},
	}
	env := config.Env{
		EventsWebhook: "",
	}

	h := NewHandler(cfg, env, nil)
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
	if h.webhookClient != nil {
		t.Error("expected webhookClient to be nil when EventsWebhook is empty")
	}

	// Close on nil webhookClient should not panic
	h.Close(context.Background())
}

func TestNewHandler_WithValidWebhook(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{Name: "Columbina"},
	}
	env := config.Env{
		EventsWebhook: "https://discord.com/api/webhooks/1234567890/tokenABC",
	}

	h := NewHandler(cfg, env, nil)
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
	if h.webhookClient == nil {
		t.Error("expected webhookClient to be non-nil for valid webhook URL")
	}

	h.Close(context.Background())
}

func TestNewHandler_WithInvalidWebhook(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{Name: "Columbina"},
	}
	env := config.Env{
		EventsWebhook: "invalid-url-not-discord",
	}

	h := NewHandler(cfg, env, nil)
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}
	if h.webhookClient != nil {
		t.Error("expected webhookClient to be nil for invalid webhook URL")
	}
}

func TestHandler_OnEvent_UnhandledType(t *testing.T) {
	h := &Handler{}
	// Calling OnEvent with an unhandled event type should safely do nothing
	readyEvent := &dgevents.Ready{}
	h.OnEvent(readyEvent)

	// Verify interface conformance
	var _ bot.EventListener = h
}
