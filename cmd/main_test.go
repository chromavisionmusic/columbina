package main

import (
	"reflect"
	"testing"

	"github.com/disgoorg/disgo/discord"
)

func TestParseEnvironment(t *testing.T) {
	t.Run("returns error when insufficient args", func(t *testing.T) {
		_, err := parseEnvironment([]string{"columbina"})
		if err == nil {
			t.Fatal("expected error when no environment argument passed")
		}
	})

	t.Run("returns environment when provided", func(t *testing.T) {
		env, err := parseEnvironment([]string{"columbina", "production"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if env != "production" {
			t.Errorf("got %q, want 'production'", env)
		}
	})
}

func TestBuildGuildInstallFallback(t *testing.T) {
	input := []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name: "test",
			IntegrationTypes: []discord.ApplicationIntegrationType{
				discord.ApplicationIntegrationTypeUserInstall,
			},
			Contexts: []discord.InteractionContextType{
				discord.InteractionContextTypePrivateChannel,
			},
		},
		discord.UserCommandCreate{
			Name: "user_cmd",
		},
	}

	result := buildGuildInstallFallback(input)
	if len(result) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(result))
	}

	slash, ok := result[0].(discord.SlashCommandCreate)
	if !ok {
		t.Fatalf("expected first command to be SlashCommandCreate, got %T", result[0])
	}

	expectedIntegrations := []discord.ApplicationIntegrationType{
		discord.ApplicationIntegrationTypeGuildInstall,
	}
	if !reflect.DeepEqual(slash.IntegrationTypes, expectedIntegrations) {
		t.Errorf("IntegrationTypes = %v, want %v", slash.IntegrationTypes, expectedIntegrations)
	}

	expectedContexts := []discord.InteractionContextType{
		discord.InteractionContextTypeGuild,
		discord.InteractionContextTypeBotDM,
	}
	if !reflect.DeepEqual(slash.Contexts, expectedContexts) {
		t.Errorf("Contexts = %v, want %v", slash.Contexts, expectedContexts)
	}

	// Verify non-slash command is preserved untouched
	userCmd, ok := result[1].(discord.UserCommandCreate)
	if !ok || userCmd.Name != "user_cmd" {
		t.Errorf("non-slash command mismatch: %+v", result[1])
	}
}
