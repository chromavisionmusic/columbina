package commands

import (
	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

var Commands = []discord.ApplicationCommandCreate{
	PingCommandMetadata,
	AboutCommandMetadata,
}

func LoadCommands(handler *handler.Mux, cfg *config.Config) {
	handler.SlashCommand("/ping", PingCommandHandler(cfg))
	handler.SlashCommand("/about", AboutCommandHandler(cfg))
}
