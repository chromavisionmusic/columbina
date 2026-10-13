package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var startTime = time.Now()

var AboutCommandMetadata = discord.SlashCommandCreate{
	Name:        "about",
	Description: "Information about columbina",
	IntegrationTypes: []discord.ApplicationIntegrationType{
		discord.ApplicationIntegrationTypeGuildInstall,
		discord.ApplicationIntegrationTypeUserInstall,
	},
	Contexts: []discord.InteractionContextType{
		discord.InteractionContextTypeGuild,
		discord.InteractionContextTypeBotDM,
		discord.InteractionContextTypePrivateChannel,
	},
}

func AboutCommandHandler(cfg *config.Config) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		embed := discord.NewEmbed().
			WithTitle(cfg.Bot.Name).
			WithColor(cfg.ColorInt()).
			WithDescription(cfg.Bot.Description).
			WithFooter("Built with disgo", "https://go.dev/blog/go-brand/Go-Logo/PNG/Go-Logo_Blue.png").
			WithTimestamp(time.Now())

		if selfUser, ok := e.Client().Caches.SelfUser(); ok {
			embed = embed.WithThumbnail(selfUser.EffectiveAvatarURL())
		}

		users := 0
		guilds := 0
		if e.Client().Caches != nil {
			guilds = e.Client().Caches.GuildsLen()
			for guild := range e.Client().Caches.Guilds() {
				users += guild.MemberCount
			}
			if users == 0 {
				users = e.Client().Caches.MembersAllLen()
			}
		}

		ping := int64(0)
		if e.Client().HasGateway() && e.Client().Gateway != nil {
			ping = e.Client().Gateway.Latency().Milliseconds()
		}

		embed = embed.
			AddField("Version", cfg.Bot.Version, true).
			AddField("Ping", fmt.Sprintf("%dms", ping), true).
			AddField("Uptime", fmt.Sprintf("<t:%d:R>", startTime.Unix()), true).
			AddField("Users", fmt.Sprintf("%d", users), true).
			AddField("Guilds", fmt.Sprintf("%d", guilds), true).
			AddField("Commands", fmt.Sprintf("%d", len(Commands)), true).
			AddField("Developed by", fmt.Sprintf("[%s](%s)", cfg.Bot.Owner.Name, fmt.Sprintf("https://discord.com/users/%s", cfg.Bot.Owner.ID)), true).
			AddField("Designed by", fmt.Sprintf("[%s](%s)", cfg.Bot.Designer.Name, fmt.Sprintf("https://discord.com/users/%s", cfg.Bot.Designer.ID)), true)

		var buttons []discord.InteractiveComponent
		if link := strings.TrimSpace(cfg.Bot.Links.Invite); link != "" {
			buttons = append(buttons, discord.NewLinkButton("Invite", link))
		}
		if link := strings.TrimSpace(cfg.Bot.Links.Website); link != "" {
			buttons = append(buttons, discord.NewLinkButton("Website", link))
		}
		if link := strings.TrimSpace(cfg.Bot.Links.Server); link != "" {
			buttons = append(buttons, discord.NewLinkButton("Support Server", link))
		}

		msg := discord.MessageCreate{
			Content: fmt.Sprintf("-# <:INFO:1559580206401069168> Interested in contributing to the project? Checkout out the [Github](%s) repo and give it a star.", cfg.Bot.Links.Github),
			Embeds:  []discord.Embed{embed},
		}
		if len(buttons) > 0 {
			msg = msg.AddActionRow(buttons...)
		}

		return e.CreateMessage(msg)
	}
}
