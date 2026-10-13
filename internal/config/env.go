package config

import (
	"log"
	"os"
	"strings"

	"github.com/disgoorg/snowflake/v2"
	"github.com/joho/godotenv"
)

type Env struct {
	Token         string
	GuildId       snowflake.ID
	ChannelId     snowflake.ID
	DatabasePath  string
	EventsWebhook string
	LogsWebhook   string
	LLMBaseUrl        string
	LLMModel          string
	LLMApiKey         string
	LLMFallbackModels []string
	TavilyApiKey      string
}

func LoadEnv(environment string) Env {
	switch environment {
	case "production", "prod":
		godotenv.Load(".env")
	case "development", "dev":
		godotenv.Load(".env.development")
	default:
		log.Fatalf("Unknown environment: %s", environment)
	}

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		panic("BOT_TOKEN is required!")
	}

	env := os.Getenv("ENV")
	if env == "" {
		panic("ENV is required!")
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = os.Getenv("DATABASE_URL")
	}
	if dbPath == "" {
		dbPath = "data/columbina.db"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	eventsWebhook := os.Getenv("BOT_EVENTS_WEBHOOK")
	if eventsWebhook == "" {
		eventsWebhook = os.Getenv("EVENTS_WEBHOOK")
	}

	logsWebhook := os.Getenv("BOT_LOGS_WEBHOOK")
	if logsWebhook == "" {
		logsWebhook = os.Getenv("LOGS_WEBHOOK")
	}

	llmUrl := os.Getenv("LLM_BASE_URL")

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		panic("LLM_MODEL is required!")
	}

	llmApiKey := os.Getenv("LLM_API_KEY")
	if llmApiKey == "" {
		panic("LLM_API_KEY is required!")
	}

	fallbackModelsRaw := os.Getenv("LLM_FALLBACK_MODELS")
	var fallbackModels []string
	if fallbackModelsRaw != "" {
		for _, m := range strings.Split(fallbackModelsRaw, ",") {
			if trimmed := strings.TrimSpace(m); trimmed != "" {
				fallbackModels = append(fallbackModels, trimmed)
			}
		}
	}

	tavilyApiKey := os.Getenv("TAVILY_API_KEY")

	return Env{
		Token:             token,
		GuildId:           snowflake.GetEnv("GUILD_ID"),
		ChannelId:         snowflake.GetEnv("CHANNEL_ID"),
		DatabasePath:      dbPath,
		EventsWebhook:     eventsWebhook,
		LogsWebhook:       logsWebhook,
		LLMBaseUrl:        llmUrl,
		LLMModel:          model,
		LLMApiKey:         llmApiKey,
		LLMFallbackModels: fallbackModels,
		TavilyApiKey:      tavilyApiKey,
	}
}
