package commands

import (
	"fmt"
	"time"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var PingCommandMetadata = discord.SlashCommandCreate{
	Name:        "ping",
	Description: "Check bot latency",
}

func PingCommandHandler(cfg *config.Config) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		apiLatency := e.Client().Gateway.Latency()
		botLatency := time.Since(e.ApplicationCommandInteraction.CreatedAt())

		embed := discord.NewEmbed().
			WithTitle("Pong!").
			WithColor(cfg.ColorInt()).
			AddField("API Latency", fmt.Sprintf("`%dms`", apiLatency.Milliseconds()), true).
			AddField("Bot Latency", fmt.Sprintf("`%dms`", botLatency.Milliseconds()), true).
			WithTimestamp(time.Now())

		return e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{embed},
		})
	}
}
