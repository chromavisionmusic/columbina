package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/log/v2"
	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/chromavisionmusic/columbina/internal/search"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/artifact"
	"google.golang.org/adk/v2/memory"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/openaimodel"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/genai"
)

// Service encapsulates the ADK Go v2 Agent, Runner, and SessionService.
type Service struct {
	runner         *runner.Runner
	sessionService session.Service
	searchClient   *search.Client
}

// NewService initializes the AI model, agent, and runner using the provided environment configuration.
func NewService(ctx context.Context, env config.Env) (*Service, error) {
	// Determine API type: default to APIResponses, support chat completions via env if specified.
	apiType := openaimodel.APIResponses
	if apiTypeEnv := strings.ToLower(os.Getenv("LLM_API_TYPE")); apiTypeEnv == "chat_completions" || apiTypeEnv == "chat" {
		apiType = openaimodel.APIChatCompletions
	}

	cfg := openaimodel.ClientConfig{
		APIKey:  env.LLMApiKey,
		BaseURL: env.LLMBaseUrl,
		API:     apiType,
	}

	var models []model.LLM
	primaryModel, err := openaimodel.NewModel(ctx, env.LLMModel, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary model %q: %w", env.LLMModel, err)
	}
	models = append(models, primaryModel)

	for _, fallbackName := range env.LLMFallbackModels {
		fbModel, err := openaimodel.NewModel(ctx, fallbackName, &cfg)
		if err != nil {
			log.Warnf("Failed to initialize fallback model %q: %v", fallbackName, err)
			continue
		}
		log.Infof("Configured fallback model %q on gateway", fallbackName)
		models = append(models, fbModel)
	}

	selectedModel := NewFallbackModel(models...)

	// Load persona/system prompt from markdown file
	promptPath := "prompts/system_prompt.md"
	systemPromptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		for _, altPath := range []string{"../../prompts/system_prompt.md", "../prompts/system_prompt.md"} {
			if b, altErr := os.ReadFile(altPath); altErr == nil {
				systemPromptBytes = b
				err = nil
				break
			}
		}
	}
	var systemPrompt string
	if err != nil {
		log.Warnf("Could not read %s, using fallback prompt: %v", promptPath, err)
		systemPrompt = "You are Columbina, a helpful, polite, and enigmatic AI assistant in Discord."
	} else {
		systemPrompt = string(systemPromptBytes)
	}

	// Initialize tools (Tavily web search)
	var tools []tool.Tool
	var searchClient *search.Client
	if strings.TrimSpace(env.TavilyApiKey) != "" {
		searchClient = search.NewClient(env.TavilyApiKey)
		searchTool, err := searchClient.AsTool()
		if err != nil {
			log.Warnf("Failed to initialize Tavily web search tool: %v", err)
		} else {
			tools = append(tools, searchTool)
			log.Info("Autonomous web search tool initialized (Tavily)")
		}
	} else {
		log.Warn("TAVILY_API_KEY is not set or empty; running with web search disabled")
	}

	columbinaAgent, err := llmagent.New(llmagent.Config{
		Name:        "columbina",
		Model:       selectedModel,
		Description: "Columbina AI chat companion for Discord.",
		Instruction: systemPrompt,
		Tools:       tools,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create llmagent: %w", err)
	}

	sessionService := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:           "columbina-bot",
		Agent:             columbinaAgent,
		SessionService:    sessionService,
		ArtifactService:   artifact.InMemoryService(),
		MemoryService:     memory.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ADK runner: %w", err)
	}

	return &Service{
		runner:         r,
		sessionService: sessionService,
		searchClient:   searchClient,
	}, nil
}

// Close releases resources held by the AI service (such as search cache cleanup).
func (s *Service) Close() {
	if s.searchClient != nil {
		s.searchClient.Close()
	}
}

// HasSearch reports whether autonomous web search is enabled.
func (s *Service) HasSearch() bool {
	return s.searchClient != nil
}

// SessionIDForGuild returns the session ID used for guild-based conversations.
func SessionIDForGuild(userID string) string {
	return fmt.Sprintf("user_%s", userID)
}

// SessionIDForDM returns the session ID used for private DM conversations.
func SessionIDForDM(userID string) string {
	return fmt.Sprintf("dm_user_%s", userID)
}

// Ask sends a prompt to the agent under a user's guild session and returns the final response.
func (s *Service) Ask(ctx context.Context, userID, prompt string) (string, error) {
	return s.AskSession(ctx, SessionIDForGuild(userID), userID, prompt)
}

// AskDM sends a prompt to the agent under a user's isolated DM session and returns the final response.
func (s *Service) AskDM(ctx context.Context, userID, prompt string) (string, error) {
	return s.AskSession(ctx, SessionIDForDM(userID), userID, prompt)
}

// AskSession sends a prompt to the agent under a specific session and returns the final response.
func (s *Service) AskSession(ctx context.Context, sessionID, userID, prompt string) (string, error) {
	userContent := genai.NewContentFromParts([]*genai.Part{
		genai.NewPartFromText(prompt),
	}, genai.RoleUser)

	var responseBuilder strings.Builder
	for event, err := range s.runner.Run(ctx, userID, sessionID, userContent, agent.RunConfig{}) {
		if err != nil {
			return "", fmt.Errorf("agent run error: %w", err)
		}

		if event.IsFinalResponse() && event.Content != nil {
			for _, part := range event.Content.Parts {
				if part.Text != "" && !part.Thought {
					responseBuilder.WriteString(part.Text)
				}
			}
		}
	}

	result := strings.TrimSpace(responseBuilder.String())
	if result == "" {
		return "...", nil
	}
	return result, nil
}

// ResetSession clears a specific session from memory.
func (s *Service) ResetSession(ctx context.Context, userID, sessionID string) error {
	if s.sessionService == nil {
		return nil
	}
	return s.sessionService.Delete(ctx, &session.DeleteRequest{
		AppName:   "columbina-bot",
		UserID:    userID,
		SessionID: sessionID,
	})
}

// ResetDMSession clears a user's DM session from memory.
func (s *Service) ResetDMSession(ctx context.Context, userID string) error {
	return s.ResetSession(ctx, userID, SessionIDForDM(userID))
}
