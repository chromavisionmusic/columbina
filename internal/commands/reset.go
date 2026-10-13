package commands

import (
	"context"
	"time"

	"github.com/chromavisionmusic/columbina/internal/ai"
	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var ResetCommandMetadata = discord.SlashCommandCreate{
	Name:        "reset",
	Description: "Reset your conversation memory with Columbina",
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

func ResetCommandHandler(cfg *config.Config, aiService *ai.Service) handler.SlashCommandHandler {
	return func(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
		userID := e.User().ID.String()

		if aiService != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_ = aiService.ResetDMSession(ctx, userID)
			_ = aiService.ResetSession(ctx, userID, ai.SessionIDForGuild(userID))
		}

		embed := discord.NewEmbed().
			WithTitle("Memory Reset").
			WithDescription("My memories of our previous conversation have gently faded into the moonlight. What shall we talk about now?").
			WithColor(cfg.ColorInt()).
			WithTimestamp(time.Now())

		return e.CreateMessage(discord.MessageCreate{
			Embeds: []discord.Embed{embed},
		})
	}
}
