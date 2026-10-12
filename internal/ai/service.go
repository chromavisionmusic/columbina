package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/log/v2"
	"github.com/chromavisionmusic/columbina/internal/config"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model/openaimodel"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// Service encapsulates the ADK Go v2 Agent and Runner.
type Service struct {
	runner *runner.Runner
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

	model, err := openaimodel.NewModel(ctx, env.LLMModel, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI model: %w", err)
	}

	// Load persona/system prompt from markdown file
	promptPath := "prompts/system_prompt.md"
	systemPromptBytes, err := os.ReadFile(promptPath)
	var systemPrompt string
	if err != nil {
		log.Warnf("Could not read %s, using fallback prompt: %v", promptPath, err)
		systemPrompt = "You are Columbina, a helpful, polite, and enigmatic AI assistant in Discord."
	} else {
		systemPrompt = string(systemPromptBytes)
	}

	columbinaAgent, err := llmagent.New(llmagent.Config{
		Name:        "columbina",
		Model:       model,
		Description: "Columbina AI chat companion for Discord.",
		Instruction: systemPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create llmagent: %w", err)
	}

	r, err := runner.NewInMemory("columbina-bot", columbinaAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to create ADK runner: %w", err)
	}

	return &Service{
		runner: r,
	}, nil
}

// Ask sends a prompt to the agent under a specific user's session and returns the final response.
func (s *Service) Ask(ctx context.Context, userID, prompt string) (string, error) {
	userContent := genai.NewContentFromParts([]*genai.Part{
		genai.NewPartFromText(prompt),
	}, genai.RoleUser)

	// Session is isolated per user
	sessionID := fmt.Sprintf("user_%s", userID)

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
