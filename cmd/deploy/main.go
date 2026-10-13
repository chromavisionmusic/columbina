package main

import (
	"context"
	"os"
	"time"

	"charm.land/log/v2"
	"github.com/chromavisionmusic/columbina/internal/commands"
	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo"
)

func resolveEnvironment(args []string) string {
	if len(args) > 1 {
		return args[1]
	}
	if env := os.Getenv("ENV"); env != "" {
		return env
	}
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		if _, err := os.Stat(".env.development"); err == nil {
			return "development"
		}
	}
	return "production"
}

func main() {
	environment := resolveEnvironment(os.Args)

	log.Infof("Deploying commands globally in %s environment...", environment)

	env := config.LoadEnv(environment)

	client, err := disgo.New(env.Token)
	if err != nil {
		log.Fatalf("Failed to initialize Discord client: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		client.Close(ctx)
	}()

	log.Infof("Publishing %d application commands globally...", len(commands.Commands))

	deployedCommands, err := client.Rest.SetGlobalCommands(client.ApplicationID, commands.Commands)
	if err != nil {
		log.Fatalf("Failed to publish global commands: %v", err)
	}

	log.Infof("Successfully deployed %d global command(s):", len(deployedCommands))
	for _, cmd := range deployedCommands {
		log.Infof("  - /%s (ID: %s)", cmd.Name(), cmd.ID())
	}
}
