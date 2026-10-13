package commands

import (
	"testing"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
)

func TestCommands_MetadataList(t *testing.T) {
	if len(Commands) != 3 {
		t.Fatalf("expected 3 commands registered, got %d", len(Commands))
	}

	expectedNames := []string{"ping", "about", "reset"}
	for i, cmd := range Commands {
		slashCmd, ok := cmd.(discord.SlashCommandCreate)
		if !ok {
			t.Fatalf("command %d is not a SlashCommandCreate: %T", i, cmd)
		}
		if slashCmd.CommandName() != expectedNames[i] {
			t.Errorf("command %d name = %q, want %q", i, slashCmd.CommandName(), expectedNames[i])
		}
	}
}

func TestLoadCommands(t *testing.T) {
	mux := handler.New()
	cfg := &config.Config{
		Bot: config.BotConfig{
			Name:    "Columbina",
			Version: "1.0.0",
		},
	}

	// Should not panic when loading commands with nil or valid aiService
	LoadCommands(mux, cfg, nil)
}
