package ai

import (
	"context"
	"errors"
	"iter"
	"strings"
	"testing"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

type mockLLM struct {
	name          string
	responses     []*model.LLMResponse
	failMidStream bool
	failErr       error
	called        bool
}

func (m *mockLLM) Name() string {
	return m.name
}

func (m *mockLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		m.called = true
		if m.failErr != nil && !m.failMidStream {
			yield(nil, m.failErr)
			return
		}

		for _, resp := range m.responses {
			if !yield(resp, nil) {
				return
			}
		}

		if m.failMidStream && m.failErr != nil {
			yield(nil, m.failErr)
		}
	}
}

func makeDummyResponse(text string) *model.LLMResponse {
	return &model.LLMResponse{
		Content: genai.NewContentFromParts([]*genai.Part{
			genai.NewPartFromText(text),
		}, genai.RoleModel),
	}
}

func TestNewFallbackModel(t *testing.T) {
	t.Run("panics on empty models", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic on empty models")
			}
		}()
		_ = NewFallbackModel()
	})

	t.Run("returns single model unwrapped", func(t *testing.T) {
		m := &mockLLM{name: "single"}
		res := NewFallbackModel(m)
		if res != m {
			t.Fatal("expected original model instance")
		}
	})

	t.Run("returns fallback model on multiple", func(t *testing.T) {
		m1 := &mockLLM{name: "m1"}
		m2 := &mockLLM{name: "m2"}
		res := NewFallbackModel(m1, m2)
		fb, ok := res.(*FallbackModel)
		if !ok {
			t.Fatalf("expected *FallbackModel, got %T", res)
		}
		if fb.Name() != "fallback[m1 -> m2]" {
			t.Errorf("Name = %q, want 'fallback[m1 -> m2]'", fb.Name())
		}
	})
}

func TestFallbackModel_PrimarySuccess(t *testing.T) {
	m1 := &mockLLM{
		name:      "primary",
		responses: []*model.LLMResponse{makeDummyResponse("hello from primary")},
	}
	m2 := &mockLLM{name: "secondary"}

	fb := NewFallbackModel(m1, m2)

	var collected []string
	var lastErr error
	for resp, err := range fb.GenerateContent(context.Background(), &model.LLMRequest{}, false) {
		if err != nil {
			lastErr = err
			break
		}
		if resp != nil && resp.Content != nil {
			collected = append(collected, resp.Content.Parts[0].Text)
		}
	}

	if lastErr != nil {
		t.Fatalf("unexpected error: %v", lastErr)
	}
	if len(collected) != 1 || collected[0] != "hello from primary" {
		t.Errorf("collected = %v, want ['hello from primary']", collected)
	}
	if !m1.called {
		t.Error("primary was not called")
	}
	if m2.called {
		t.Error("secondary should NOT have been called")
	}
}

func TestFallbackModel_FailoverToSecondary(t *testing.T) {
	m1 := &mockLLM{
		name:    "primary",
		failErr: errors.New("primary rate limit"),
	}
	m2 := &mockLLM{
		name:      "secondary",
		responses: []*model.LLMResponse{makeDummyResponse("hello from secondary")},
	}

	fb := NewFallbackModel(m1, m2)

	var collected []string
	var lastErr error
	for resp, err := range fb.GenerateContent(context.Background(), &model.LLMRequest{}, false) {
		if err != nil {
			lastErr = err
			break
		}
		if resp != nil && resp.Content != nil {
			collected = append(collected, resp.Content.Parts[0].Text)
		}
	}

	if lastErr != nil {
		t.Fatalf("unexpected error: %v", lastErr)
	}
	if !m1.called || !m2.called {
		t.Errorf("expected both models called: m1=%v, m2=%v", m1.called, m2.called)
	}
	if len(collected) != 1 || collected[0] != "hello from secondary" {
		t.Errorf("collected = %v, want ['hello from secondary']", collected)
	}
}

func TestFallbackModel_MidStreamFailureDoesNotFailover(t *testing.T) {
	m1 := &mockLLM{
		name:          "primary",
		responses:     []*model.LLMResponse{makeDummyResponse("partial response")},
		failMidStream: true,
		failErr:       errors.New("stream disconnect"),
	}
	m2 := &mockLLM{name: "secondary"}

	fb := NewFallbackModel(m1, m2)

	var collected []string
	var caughtErr error
	for resp, err := range fb.GenerateContent(context.Background(), &model.LLMRequest{}, false) {
		if err != nil {
			caughtErr = err
			break
		}
		if resp != nil && resp.Content != nil {
			collected = append(collected, resp.Content.Parts[0].Text)
		}
	}

	if caughtErr == nil || !strings.Contains(caughtErr.Error(), "stream disconnect") {
		t.Fatalf("expected stream disconnect error, got %v", caughtErr)
	}
	if len(collected) != 1 || collected[0] != "partial response" {
		t.Errorf("expected 1 partial response, got %v", collected)
	}
	if m2.called {
		t.Error("secondary should NOT be called after partial yield")
	}
}

func TestFallbackModel_AllModelsFail(t *testing.T) {
	m1 := &mockLLM{name: "m1", failErr: errors.New("m1 error")}
	m2 := &mockLLM{name: "m2", failErr: errors.New("m2 error")}

	fb := NewFallbackModel(m1, m2)

	var caughtErr error
	for _, err := range fb.GenerateContent(context.Background(), nil, false) {
		if err != nil {
			caughtErr = err
			break
		}
	}

	if caughtErr == nil || !strings.Contains(caughtErr.Error(), "all models in fallback chain failed") {
		t.Fatalf("expected all models failed error, got %v", caughtErr)
	}
}

func TestFallbackModel_EarlyBreak(t *testing.T) {
	m1 := &mockLLM{
		name: "m1",
		responses: []*model.LLMResponse{
			makeDummyResponse("part 1"),
			makeDummyResponse("part 2"),
		},
	}
	fb := NewFallbackModel(m1)

	var count int
	for range fb.GenerateContent(context.Background(), &model.LLMRequest{}, false) {
		count++
		break // Break after first item
	}

	if count != 1 {
		t.Errorf("expected count 1 after early break, got %d", count)
	}
}
