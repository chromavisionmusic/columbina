package events

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	dgevents "github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/disgo/webhook"
	"github.com/disgoorg/snowflake/v2"
)

func TestBuildGuildJoinEmbed_NilConfigAndBasic(t *testing.T) {
	guild := discord.GatewayGuild{
		RestGuild: discord.RestGuild{
			Guild: discord.Guild{
				ID:          snowflake.New(time.Now()),
				Name:        "Basic Guild",
				OwnerID:     snowflake.ID(123456789),
				MemberCount: 42,
			},
		},
	}

	embed := BuildGuildJoinEmbed(nil, guild, 5, 0, "")

	if embed.Title != "Joined a new community!" {
		t.Errorf("Title = %q, want 'Joined a new community!'", embed.Title)
	}
	if embed.Color != config.DefaultEmbedColor {
		t.Errorf("Color = %x, want %x", embed.Color, config.DefaultEmbedColor)
	}
	if embed.Footer == nil || !strings.Contains(embed.Footer.Text, "columbina • Guild #5") {
		t.Errorf("Footer = %+v", embed.Footer)
	}

	fieldsMap := make(map[string]string)
	for _, f := range embed.Fields {
		fieldsMap[f.Name] = f.Value
	}

	if fieldsMap["Server Name"] != "Basic Guild" {
		t.Errorf("Server Name = %q", fieldsMap["Server Name"])
	}
	if fieldsMap["Members"] != "42" {
		t.Errorf("Members = %q", fieldsMap["Members"])
	}
	if fieldsMap["Total Servers"] != "5" {
		t.Errorf("Total Servers = %q", fieldsMap["Total Servers"])
	}
	if _, ok := fieldsMap["Vanity URL"]; ok {
		t.Error("expected no Vanity URL field")
	}
	if _, ok := fieldsMap["Total Users"]; ok {
		t.Error("expected no Total Users field when totalUsers is 0")
	}
}

func TestBuildGuildJoinEmbed_Full(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{
			Name:  "ColumbinaBot",
			Color: 0x9c3f73,
		},
	}

	desc := "A lovely server for music lovers"
	vanity := "music-hangout"
	iconHash := "iconhash123"
	bannerHash := "bannerhash456"

	guild := discord.GatewayGuild{
		RestGuild: discord.RestGuild{
			Guild: discord.Guild{
				ID:            snowflake.ID(111222333),
				Name:          "Snezhnaya Realm",
				OwnerID:       snowflake.ID(999888777),
				MemberCount:   1500,
				Description:   &desc,
				VanityURLCode: &vanity,
				Icon:          &iconHash,
				Banner:        &bannerHash,
			},
		},
	}

	avatarURL := "https://cdn.discordapp.com/avatars/bot.png"
	embed := BuildGuildJoinEmbed(cfg, guild, 10, 8500, avatarURL)

	if embed.Color != 0x9c3f73 {
		t.Errorf("Color = %x, want 0x9c3f73", embed.Color)
	}
	if embed.Description != desc {
		t.Errorf("Description = %q, want %q", embed.Description, desc)
	}
	if embed.Thumbnail == nil {
		t.Error("expected thumbnail to be populated")
	}
	if embed.Footer == nil || embed.Footer.IconURL != avatarURL || !strings.Contains(embed.Footer.Text, "ColumbinaBot • Guild #10") {
		t.Errorf("Footer mismatch: %+v", embed.Footer)
	}

	fieldsMap := make(map[string]string)
	for _, f := range embed.Fields {
		fieldsMap[f.Name] = f.Value
	}

	if fieldsMap["Vanity URL"] != "https://discord.gg/music-hangout" {
		t.Errorf("Vanity URL = %q", fieldsMap["Vanity URL"])
	}
	if fieldsMap["Total Users"] != "8500" {
		t.Errorf("Total Users = %q", fieldsMap["Total Users"])
	}
}

func TestBuildWebhookMessage(t *testing.T) {
	cfg := &config.Config{
		Bot: config.BotConfig{
			Name: "Columbina",
		},
	}
	guild := discord.GatewayGuild{
		RestGuild: discord.RestGuild{
			Guild: discord.Guild{
				ID:   snowflake.ID(123),
				Name: "Guild",
			},
		},
	}

	avatar := "https://example.com/avatar.png"
	msg := BuildWebhookMessage(cfg, guild, 3, 100, avatar)

	if msg.Username != "Columbina" {
		t.Errorf("Username = %q, want 'Columbina'", msg.Username)
	}
	if msg.AvatarURL != avatar {
		t.Errorf("AvatarURL = %q, want %q", msg.AvatarURL, avatar)
	}
	if len(msg.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(msg.Embeds))
	}
}

func TestHandler_OnGuildJoin_NilWebhook(t *testing.T) {
	h := &Handler{
		webhookClient: nil,
	}

	// Should safely return early without panic
	event := &dgevents.GuildJoin{
		Guild: discord.GatewayGuild{
			RestGuild: discord.RestGuild{
				Guild: discord.Guild{
					ID:   snowflake.ID(123),
					Name: "Guild",
				},
			},
		},
	}
	h.OnGuildJoin(event)
}

func TestHandler_OnGuildJoin_WithWebhookSuccess(t *testing.T) {
	capturedBody := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		capturedBody <- body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1","type":1,"channel_id":"2"}`))
	}))
	defer srv.Close()

	webhookClient := webhook.New(snowflake.ID(12345), "mock-token", webhook.WithRestClientConfigOpts(rest.WithURL(srv.URL)))
	cfg := &config.Config{
		Bot: config.BotConfig{
			Name:  "ColumbinaBot",
			Color: 0x9c3f73,
		},
	}
	h := &Handler{
		cfg:           cfg,
		webhookClient: webhookClient,
	}

	c := cache.New(cache.WithCaches(cache.FlagGuilds, cache.FlagMembers))
	avatar := "self-avatar-hash"
	c.SetSelfUser(discord.OAuth2User{
		User: discord.User{
			ID:     snowflake.ID(9001),
			Avatar: &avatar,
		},
	})
	c.GuildCache().Put(snowflake.ID(101), discord.Guild{
		ID:          snowflake.ID(101),
		MemberCount: 50,
	})
	c.GuildCache().Put(snowflake.ID(102), discord.Guild{
		ID:          snowflake.ID(102),
		MemberCount: 150,
	})

	client := &bot.Client{Caches: c}
	genericEvent := dgevents.NewGenericEvent(client, 1, 0)
	genericGuild := &dgevents.GenericGuild{
		GenericEvent: genericEvent,
		GuildID:      snowflake.ID(101),
	}

	event := &dgevents.GuildJoin{
		GenericGuild: genericGuild,
		Guild: discord.GatewayGuild{
			RestGuild: discord.RestGuild{
				Guild: discord.Guild{
					ID:          snowflake.ID(101),
					Name:        "New Guild",
					MemberCount: 50,
				},
			},
		},
	}

	h.OnGuildJoin(event)

	select {
	case body := <-capturedBody:
		var webhookMsg discord.WebhookMessageCreate
		if err := json.Unmarshal(body, &webhookMsg); err != nil {
			t.Fatalf("failed to unmarshal webhook payload: %v", err)
		}
		if webhookMsg.Username != "ColumbinaBot" {
			t.Errorf("Username = %q, want 'ColumbinaBot'", webhookMsg.Username)
		}
		if len(webhookMsg.Embeds) != 1 {
			t.Fatalf("expected 1 embed, got %d", len(webhookMsg.Embeds))
		}
		embed := webhookMsg.Embeds[0]
		if embed.Title != "Joined a new community!" {
			t.Errorf("embed.Title = %q, want 'Joined a new community!'", embed.Title)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook HTTP request")
	}
}

func TestHandler_OnGuildJoin_WithWebhookError(t *testing.T) {
	requestReceived := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived <- struct{}{}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"internal error","code":0}`))
	}))
	defer srv.Close()

	webhookClient := webhook.New(snowflake.ID(12345), "mock-token", webhook.WithRestClientConfigOpts(rest.WithURL(srv.URL)))
	h := &Handler{
		cfg:           &config.Config{},
		webhookClient: webhookClient,
	}

	event := &dgevents.GuildJoin{
		Guild: discord.GatewayGuild{
			RestGuild: discord.RestGuild{
				Guild: discord.Guild{
					ID:   snowflake.ID(200),
					Name: "Error Guild",
				},
			},
		},
	}

	h.OnGuildJoin(event)

	select {
	case <-requestReceived:
		// Succeeded in exercising error logging path safely
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook error attempt")
	}
}
