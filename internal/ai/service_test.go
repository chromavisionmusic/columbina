package ai

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/chromavisionmusic/columbina/internal/config"
	"github.com/chromavisionmusic/columbina/internal/search"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/artifact"
	"google.golang.org/adk/v2/memory"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

func TestSessionIDHelpers(t *testing.T) {
	if got := SessionIDForGuild("12345"); got != "user_12345" {
		t.Errorf("SessionIDForGuild = %q, want 'user_12345'", got)
	}

	if got := SessionIDForDM("67890"); got != "dm_user_67890" {
		t.Errorf("SessionIDForDM = %q, want 'dm_user_67890'", got)
	}
}

func TestService_CloseAndHasSearch(t *testing.T) {
	sNoSearch := &Service{}
	if sNoSearch.HasSearch() {
		t.Error("expected HasSearch == false for nil searchClient")
	}
	sNoSearch.Close() // Should not panic

	sWithSearch := &Service{
		searchClient: search.NewClient("fake-key"),
	}
	if !sWithSearch.HasSearch() {
		t.Error("expected HasSearch == true for valid searchClient")
	}
	sWithSearch.Close() // Should close search client
}

func TestService_ResetSession(t *testing.T) {
	ctx := context.Background()

	t.Run("nil sessionService returns nil", func(t *testing.T) {
		s := &Service{}
		if err := s.ResetSession(ctx, "u1", "s1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := s.ResetDMSession(ctx, "u1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("active sessionService deletes session", func(t *testing.T) {
		sessService := session.InMemoryService()
		s := &Service{sessionService: sessService}

		// Create a session first
		_, err := sessService.Create(ctx, &session.CreateRequest{
			AppName:   "columbina-bot",
			UserID:    "user123",
			SessionID: SessionIDForDM("user123"),
		})
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		// Reset session
		if err := s.ResetDMSession(ctx, "user123"); err != nil {
			t.Fatalf("ResetDMSession failed: %v", err)
		}

		// Verify deletion
		getResp, err := sessService.Get(ctx, &session.GetRequest{
			AppName:   "columbina-bot",
			UserID:    "user123",
			SessionID: SessionIDForDM("user123"),
		})
		if err == nil && getResp != nil && getResp.Session != nil {
			t.Errorf("expected session to be deleted, but still found: %+v", getResp.Session)
		}
	})
}

type controllableLLM struct {
	name      string
	responses []*model.LLMResponse
	err       error
}

func (c *controllableLLM) Name() string {
	return c.name
}

func (c *controllableLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		if c.err != nil {
			yield(nil, c.err)
			return
		}
		for _, r := range c.responses {
			if !yield(r, nil) {
				return
			}
		}
	}
}

func createTestService(t *testing.T, llm model.LLM) *Service {
	t.Helper()
	columbinaAgent, err := llmagent.New(llmagent.Config{
		Name:        "columbina",
		Model:       llm,
		Description: "test agent",
		Instruction: "test instruction",
	})
	if err != nil {
		t.Fatalf("failed to create llmagent: %v", err)
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
		t.Fatalf("failed to create runner: %v", err)
	}

	return &Service{
		runner:         r,
		sessionService: sessionService,
	}
}

func TestService_AskMethods(t *testing.T) {
	ctx := context.Background()

	t.Run("normal response via Ask and AskDM", func(t *testing.T) {
		llm := &controllableLLM{
			name: "mock",
			responses: []*model.LLMResponse{
				{
					Content: genai.NewContentFromParts([]*genai.Part{
						genai.NewPartFromText("Hello traveler!"),
					}, genai.RoleModel),
				},
			},
		}

		s := createTestService(t, llm)

		reply, err := s.Ask(ctx, "u1", "Hi")
		if err != nil {
			t.Fatalf("Ask failed: %v", err)
		}
		if reply != "Hello traveler!" {
			t.Errorf("reply = %q, want 'Hello traveler!'", reply)
		}

		replyDM, err := s.AskDM(ctx, "u1", "Hi in DM")
		if err != nil {
			t.Fatalf("AskDM failed: %v", err)
		}
		if replyDM != "Hello traveler!" {
			t.Errorf("replyDM = %q, want 'Hello traveler!'", replyDM)
		}
	})

	t.Run("filters out thought parts", func(t *testing.T) {
		llm := &controllableLLM{
			name: "mock",
			responses: []*model.LLMResponse{
				{
					Content: genai.NewContentFromParts([]*genai.Part{
						{Text: "internal reasoning", Thought: true},
						genai.NewPartFromText("Visible message"),
					}, genai.RoleModel),
				},
			},
		}

		s := createTestService(t, llm)

		reply, err := s.AskSession(ctx, "sess-1", "u1", "test")
		if err != nil {
			t.Fatalf("AskSession failed: %v", err)
		}
		if reply != "Visible message" {
			t.Errorf("expected thought excluded, got %q", reply)
		}
	})

	t.Run("empty content returns ellipsis", func(t *testing.T) {
		llm := &controllableLLM{
			name: "mock",
			responses: []*model.LLMResponse{
				{
					Content: genai.NewContentFromParts([]*genai.Part{
						{Text: "only thought", Thought: true},
					}, genai.RoleModel),
				},
			},
		}

		s := createTestService(t, llm)

		reply, err := s.Ask(ctx, "u2", "silent prompt")
		if err != nil {
			t.Fatalf("Ask failed: %v", err)
		}
		if reply != "..." {
			t.Errorf("reply = %q, want '...'", reply)
		}
	})

	t.Run("runner error propagated", func(t *testing.T) {
		llm := &controllableLLM{
			name: "mock",
			err:  errors.New("inference crash"),
		}

		s := createTestService(t, llm)

		_, err := s.Ask(ctx, "u3", "fail prompt")
		if err == nil || !strings.Contains(err.Error(), "agent run error") {
			t.Fatalf("expected agent run error, got %v", err)
		}
	})
}

func TestNewService_Initialization(t *testing.T) {
	ctx := context.Background()

	t.Run("initialization with search and fallbacks", func(t *testing.T) {
		t.Setenv("LLM_API_TYPE", "chat_completions")

		env := config.Env{
			LLMApiKey:         "test-key",
			LLMBaseUrl:        "https://api.example.com",
			LLMModel:          "gpt-4o",
			LLMFallbackModels: []string{"gpt-3.5-turbo"},
			TavilyApiKey:      "tvly-test-key",
		}

		svc, err := NewService(ctx, env)
		if err != nil {
			t.Fatalf("NewService failed: %v", err)
		}
		defer svc.Close()

		if !svc.HasSearch() {
			t.Error("expected search to be enabled when TavilyApiKey is set")
		}
	})

	t.Run("initialization without search", func(t *testing.T) {
		t.Setenv("LLM_API_TYPE", "")

		env := config.Env{
			LLMApiKey:         "test-key",
			LLMBaseUrl:        "https://api.example.com",
			LLMModel:          "gpt-4o",
			LLMFallbackModels: nil,
			TavilyApiKey:      "",
		}

		svc, err := NewService(ctx, env)
		if err != nil {
			t.Fatalf("NewService failed: %v", err)
		}
		defer svc.Close()

		if svc.HasSearch() {
			t.Error("expected search to be disabled when TavilyApiKey is empty")
		}
	})
}

func TestMockLLM(t *testing.T) {
	mDefault := &MockLLM{}
	if mDefault.Name() != "mock-llm" {
		t.Errorf("expected 'mock-llm', got %q", mDefault.Name())
	}

	mNamed := &MockLLM{ModelName: "custom-model"}
	if mNamed.Name() != "custom-model" {
		t.Errorf("expected 'custom-model', got %q", mNamed.Name())
	}
}

func TestNewTestService(t *testing.T) {
	llm := &MockLLM{ModelName: "test-model"}
	sess := session.InMemoryService()
	svc, err := NewTestService(llm, sess)
	if err != nil {
		t.Fatalf("NewTestService failed: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewService_FallbackSystemPrompt(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()
	env := config.Env{
		LLMApiKey:  "test-key",
		LLMBaseUrl: "https://api.example.com",
		LLMModel:   "gpt-4o",
	}

	svc, err := NewService(ctx, env)
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	defer svc.Close()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}
