package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"charm.land/log/v2"
	"github.com/chromavisionmusic/columbina/internal/ai"
	"github.com/chromavisionmusic/columbina/internal/commands"
	"github.com/chromavisionmusic/columbina/internal/config"
	botEvents "github.com/chromavisionmusic/columbina/internal/events"
	"github.com/chromavisionmusic/columbina/internal/utils"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
)

func parseEnvironment(args []string) (string, error) {
	if len(args) < 2 {
		return "", fmt.Errorf("environment not specified")
	}
	return args[1], nil
}

func buildGuildInstallFallback(cmds []discord.ApplicationCommandCreate) []discord.ApplicationCommandCreate {
	fallbackCmds := make([]discord.ApplicationCommandCreate, len(cmds))
	for i, c := range cmds {
		if slash, ok := c.(discord.SlashCommandCreate); ok {
			slash.IntegrationTypes = []discord.ApplicationIntegrationType{
				discord.ApplicationIntegrationTypeGuildInstall,
			}
			slash.Contexts = []discord.InteractionContextType{
				discord.InteractionContextTypeGuild,
				discord.InteractionContextTypeBotDM,
			}
			fallbackCmds[i] = slash
		} else {
			fallbackCmds[i] = c
		}
	}
	return fallbackCmds
}

func main() {
	environment, err := parseEnvironment(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	env := config.LoadEnv(environment)
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatal(err)
	}

	// Command handler
	cmd := handler.New()

	// AI service
	aiCtx := context.Background()
	aiService, err := ai.NewService(aiCtx, env)
	if err != nil {
		log.Fatalf("Failed to initialize AI service: %v", err)
	}

	// Event handler
	eventHandler := botEvents.NewHandler(cfg, env, aiService)

	// Create a new client instance
	client, err := disgo.New(env.Token,
		bot.WithEventListeners(cmd, eventHandler),
		// Set gateway configuration options
		bot.WithGatewayConfigOpts(
			// Set enabled intents
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildMessages,
				gateway.IntentDirectMessages,
				gateway.IntentMessageContent,
			),
			// Set presence from config
			gateway.WithPresenceOpts(utils.GetPresenceOpts(cfg.Bot.Presence)...),
		),
		// Enable guild caching so guild list and member counts are stored
		bot.WithCacheConfigOpts(
			cache.WithCaches(cache.FlagGuilds, cache.FlagMembers),
		),
		// Listen to the Ready event in order to know when the bot is connected
		bot.WithEventListenerFunc(func(e *events.Ready) {
			log.Infof("Bot is connected as %s", e.User.Username)
		}),
	)
	if err != nil {
		panic(err)
	}

	commands.LoadCommands(cmd, cfg, aiService)

	// In Discord, slash commands in DMs MUST be registered as Global Commands.
	// Guild commands (synced to a specific server) can NEVER be accessed in DMs.
	// Therefore, we sync commands globally so they work in both DMs and servers.
	if err := handler.SyncCommands(client, commands.Commands, nil); err != nil {
		log.Warnf("Failed to sync global commands with UserInstall; retrying with GuildInstall: %v", err)
		fallbackCmds := buildGuildInstallFallback(commands.Commands)
		if errFallback := handler.SyncCommands(client, fallbackCmds, nil); errFallback != nil {
			log.Errorf("Failed to sync global commands: %v", errFallback)
		} else {
			log.Infof("Synced %d global commands with GuildInstall fallback", len(fallbackCmds))
		}
	} else {
		log.Infof("Synced %d global commands", len(commands.Commands))
	}

	// In development, clean up old guild-scoped commands so duplicates don't appear
	if (environment == "development" || environment == "dev") && env.GuildId != 0 {
		_ = handler.SyncCommands(client, []discord.ApplicationCommandCreate{}, []snowflake.ID{env.GuildId})
	}

	// Ensure we close the client on exit
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		aiService.Close()
		eventHandler.Close(ctx)
		client.Close(ctx)
	}()

	// Connect to the gateway
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = client.OpenGateway(ctx); err != nil {
		panic(err)
	}

	// Wait here until CTRL+C or other term signal is received.
	log.Info("Bot is now running. Press CTRL+C to exit.")
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
	<-s
	log.Info("Shutting down bot...")
}
