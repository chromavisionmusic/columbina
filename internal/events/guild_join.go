package events

import (
	"context"
	"fmt"
	"time"

	"charm.land/log/v2"
	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	dgevents "github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/rest"
)

func BuildGuildJoinEmbed(cfg *config.Config, guild discord.GatewayGuild, totalGuilds int, totalUsers int, avatarURL string) discord.Embed {
	botName := "columbina"
	if cfg != nil && cfg.Bot.Name != "" {
		botName = cfg.Bot.Name
	}

	embed := discord.NewEmbed().
		WithTitle("Joined a new community!").
		WithColor(cfg.ColorInt()).
		WithTimestamp(time.Now())

	if guild.Description != nil && *guild.Description != "" {
		embed = embed.WithDescription(*guild.Description)
	}

	if iconURL := guild.IconURL(); iconURL != nil {
		embed = embed.WithThumbnail(*iconURL)
	}

	if bannerURL := guild.BannerURL(); bannerURL != nil {
		embed = embed.WithThumbnail(*bannerURL)
	}

	embed = embed.
		AddField("Server Name", guild.Name, true).
		AddField("Server ID", fmt.Sprintf("`%s`", guild.ID), true).
		AddField("Owner", fmt.Sprintf("<@%s> (`%s`)", guild.OwnerID, guild.OwnerID), true).
		AddField("Members", fmt.Sprintf("%d", guild.MemberCount), true).
		AddField("Created At", fmt.Sprintf("<t:%d:R>", guild.CreatedAt().Unix()), true).
		AddField("Total Servers", fmt.Sprintf("%d", totalGuilds), true)

	if guild.VanityURLCode != nil && *guild.VanityURLCode != "" {
		embed = embed.AddField("Vanity URL", fmt.Sprintf("https://discord.gg/%s", *guild.VanityURLCode), true)
	}

	if totalUsers > 0 {
		embed = embed.AddField("Total Users", fmt.Sprintf("%d", totalUsers), true)
	}

	footerText := fmt.Sprintf("%s • Guild #%d", botName, totalGuilds)
	if avatarURL != "" {
		embed = embed.WithFooter(footerText, avatarURL)
	} else {
		embed = embed.WithFooter(footerText, "")
	}

	return embed
}

func BuildWebhookMessage(cfg *config.Config, guild discord.GatewayGuild, totalGuilds int, totalUsers int, avatarURL string) discord.WebhookMessageCreate {
	botName := "columbina"
	if cfg != nil && cfg.Bot.Name != "" {
		botName = cfg.Bot.Name
	}

	embed := BuildGuildJoinEmbed(cfg, guild, totalGuilds, totalUsers, avatarURL)

	msg := discord.WebhookMessageCreate{
		Embeds: []discord.Embed{embed},
	}
	if botName != "" {
		msg.Username = botName
	}
	if avatarURL != "" {
		msg.AvatarURL = avatarURL
	}

	return msg
}

func (h *Handler) OnGuildJoin(e *dgevents.GuildJoin) {
	guild := e.Guild

	totalGuilds := 0
	totalUsers := 0
	if e.GenericGuild != nil && e.GenericEvent != nil && e.Client() != nil && e.Client().Caches != nil {
		totalGuilds = e.Client().Caches.GuildsLen()
		for g := range e.Client().Caches.Guilds() {
			totalUsers += g.MemberCount
		}
		if totalUsers == 0 {
			totalUsers = e.Client().Caches.MembersAllLen()
		}
	}
	if totalGuilds == 0 {
		totalGuilds = 1
	}

	log.Infof("Bot joined guild: %s (ID: %s, Members: %d, Total Guilds: %d)",
		guild.Name, guild.ID, guild.MemberCount, totalGuilds)

	if h.webhookClient == nil {
		return
	}

	var avatarURL string
	if e.GenericGuild != nil && e.GenericEvent != nil && e.Client() != nil && e.Client().Caches != nil {
		if selfUser, ok := e.Client().Caches.SelfUser(); ok {
			avatarURL = selfUser.EffectiveAvatarURL()
		}
	}

	msg := BuildWebhookMessage(h.cfg, guild, totalGuilds, totalUsers, avatarURL)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := h.webhookClient.CreateMessage(msg, rest.CreateWebhookMessageParams{}, rest.WithCtx(ctx))
		if err != nil {
			log.Errorf("Failed to send guild join event to webhook: %v", err)
		}
	}()
}
