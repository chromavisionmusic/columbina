package config

import (
	"reflect"
	"testing"

	"github.com/disgoorg/snowflake/v2"
)

func setRequiredEnvs(t *testing.T) {
	t.Helper()
	t.Setenv("BOT_TOKEN", "test-bot-token")
	t.Setenv("ENV", "development")
	t.Setenv("LLM_MODEL", "gemini-2.5-flash")
	t.Setenv("LLM_API_KEY", "test-llm-key")
}

func TestLoadEnv_Success(t *testing.T) {
	setRequiredEnvs(t)
	t.Setenv("GUILD_ID", "123456789012345678")
	t.Setenv("CHANNEL_ID", "987654321098765432")
	t.Setenv("DATABASE_PATH", "/custom/path.db")
	t.Setenv("BOT_EVENTS_WEBHOOK", "https://discord.com/api/webhooks/1")
	t.Setenv("BOT_LOGS_WEBHOOK", "https://discord.com/api/webhooks/2")
	t.Setenv("LLM_BASE_URL", "https://api.example.com/v1")
	t.Setenv("LLM_FALLBACK_MODELS", "model-a, model-b , , model-c")
	t.Setenv("TAVILY_API_KEY", "tvly-test-key")

	env := LoadEnv("development")

	if env.Token != "test-bot-token" {
		t.Errorf("Token = %q, want %q", env.Token, "test-bot-token")
	}
	if env.GuildId != snowflake.ID(123456789012345678) {
		t.Errorf("GuildId = %v, want 123456789012345678", env.GuildId)
	}
	if env.ChannelId != snowflake.ID(987654321098765432) {
		t.Errorf("ChannelId = %v, want 987654321098765432", env.ChannelId)
	}
	if env.DatabasePath != "/custom/path.db" {
		t.Errorf("DatabasePath = %q, want %q", env.DatabasePath, "/custom/path.db")
	}
	if env.EventsWebhook != "https://discord.com/api/webhooks/1" {
		t.Errorf("EventsWebhook = %q", env.EventsWebhook)
	}
	if env.LogsWebhook != "https://discord.com/api/webhooks/2" {
		t.Errorf("LogsWebhook = %q", env.LogsWebhook)
	}
	if env.LLMBaseUrl != "https://api.example.com/v1" {
		t.Errorf("LLMBaseUrl = %q", env.LLMBaseUrl)
	}
	if env.LLMModel != "gemini-2.5-flash" {
		t.Errorf("LLMModel = %q", env.LLMModel)
	}
	if env.LLMApiKey != "test-llm-key" {
		t.Errorf("LLMApiKey = %q", env.LLMApiKey)
	}
	expectedFallbacks := []string{"model-a", "model-b", "model-c"}
	if !reflect.DeepEqual(env.LLMFallbackModels, expectedFallbacks) {
		t.Errorf("LLMFallbackModels = %v, want %v", env.LLMFallbackModels, expectedFallbacks)
	}
	if env.TavilyApiKey != "tvly-test-key" {
		t.Errorf("TavilyApiKey = %q", env.TavilyApiKey)
	}
}

func TestLoadEnv_ProductionAlias(t *testing.T) {
	setRequiredEnvs(t)
	env := LoadEnv("prod")
	if env.Token != "test-bot-token" {
		t.Errorf("expected token loaded, got %q", env.Token)
	}

	env2 := LoadEnv("production")
	if env2.Token != "test-bot-token" {
		t.Errorf("expected token loaded, got %q", env2.Token)
	}

	env3 := LoadEnv("dev")
	if env3.Token != "test-bot-token" {
		t.Errorf("expected token loaded, got %q", env3.Token)
	}
}

func TestLoadEnv_DatabasePathFallbacks(t *testing.T) {
	t.Run("default db path when unset", func(t *testing.T) {
		setRequiredEnvs(t)
		t.Setenv("DATABASE_PATH", "")
		t.Setenv("DATABASE_URL", "")

		env := LoadEnv("dev")
		if env.DatabasePath != "data/columbina.db" {
			t.Errorf("expected default db path 'data/columbina.db', got %q", env.DatabasePath)
		}
	})

	t.Run("DATABASE_URL fallback when DATABASE_PATH empty", func(t *testing.T) {
		setRequiredEnvs(t)
		t.Setenv("DATABASE_PATH", "")
		t.Setenv("DATABASE_URL", "sqlite:///url.db")

		env := LoadEnv("dev")
		if env.DatabasePath != "sqlite:///url.db" {
			t.Errorf("expected DATABASE_URL fallback, got %q", env.DatabasePath)
		}
	})

	t.Run("DATABASE_PATH takes precedence over DATABASE_URL", func(t *testing.T) {
		setRequiredEnvs(t)
		t.Setenv("DATABASE_PATH", "/explicit.db")
		t.Setenv("DATABASE_URL", "sqlite:///url.db")

		env := LoadEnv("dev")
		if env.DatabasePath != "/explicit.db" {
			t.Errorf("expected /explicit.db, got %q", env.DatabasePath)
		}
	})
}

func TestLoadEnv_WebhookFallbacks(t *testing.T) {
	t.Run("fallback webhook names", func(t *testing.T) {
		setRequiredEnvs(t)
		t.Setenv("BOT_EVENTS_WEBHOOK", "")
		t.Setenv("EVENTS_WEBHOOK", "https://discord.com/events")
		t.Setenv("BOT_LOGS_WEBHOOK", "")
		t.Setenv("LOGS_WEBHOOK", "https://discord.com/logs")

		env := LoadEnv("dev")
		if env.EventsWebhook != "https://discord.com/events" {
			t.Errorf("EventsWebhook = %q", env.EventsWebhook)
		}
		if env.LogsWebhook != "https://discord.com/logs" {
			t.Errorf("LogsWebhook = %q", env.LogsWebhook)
		}
	})
}

func TestLoadEnv_Panics(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T)
		panicSubstr string
	}{
		{
			name: "missing BOT_TOKEN",
			setup: func(t *testing.T) {
				setRequiredEnvs(t)
				t.Setenv("BOT_TOKEN", "")
			},
			panicSubstr: "BOT_TOKEN is required!",
		},
		{
			name: "missing ENV",
			setup: func(t *testing.T) {
				setRequiredEnvs(t)
				t.Setenv("ENV", "")
			},
			panicSubstr: "ENV is required!",
		},
		{
			name: "missing LLM_MODEL",
			setup: func(t *testing.T) {
				setRequiredEnvs(t)
				t.Setenv("LLM_MODEL", "")
			},
			panicSubstr: "LLM_MODEL is required!",
		},
		{
			name: "missing LLM_API_KEY",
			setup: func(t *testing.T) {
				setRequiredEnvs(t)
				t.Setenv("LLM_API_KEY", "")
			},
			panicSubstr: "LLM_API_KEY is required!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("expected panic %q, got nil", tt.panicSubstr)
				}
				msg, ok := r.(string)
				if !ok || msg != tt.panicSubstr {
					t.Fatalf("expected panic message %q, got %v", tt.panicSubstr, r)
				}
			}()

			LoadEnv("dev")
		})
	}
}
