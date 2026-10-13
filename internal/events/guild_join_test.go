package events

import (
	"strings"
	"testing"
	"time"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	dgevents "github.com/disgoorg/disgo/events"
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
