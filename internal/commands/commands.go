package commands

import (
	"github.com/chromavisionmusic/columbina/internal/ai"
	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var Commands = []discord.ApplicationCommandCreate{
	PingCommandMetadata,
	AboutCommandMetadata,
	ResetCommandMetadata,
}

func LoadCommands(handler *handler.Mux, cfg *config.Config, aiService *ai.Service) {
	handler.SlashCommand("/ping", PingCommandHandler(cfg))
	handler.SlashCommand("/about", AboutCommandHandler(cfg))
	handler.SlashCommand("/reset", ResetCommandHandler(cfg, aiService))
}
