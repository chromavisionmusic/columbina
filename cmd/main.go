package main

import (
	"context"
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
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/snowflake/v2"
)

func main() {
	var environment string

	if len(os.Args) < 2 {
		log.Fatal("Environment not specified")
	} else {
		environment = os.Args[1]
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

	commands.LoadCommands(cmd, cfg)

	var guildIDs []snowflake.ID
	if environment == "development" || environment == "dev" {
		guildIDs = []snowflake.ID{env.GuildId}
	}

	if err := handler.SyncCommands(client, commands.Commands, guildIDs); err != nil {
		panic(err)
	}
	log.Infof("Synced %d commands", len(commands.Commands))

	// Ensure we close the client on exit
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
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
