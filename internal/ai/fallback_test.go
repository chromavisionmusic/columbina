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

// mockModel implements model.LLM for testing
type mockModel struct {
	name      string
	shouldErr bool
	response  string
	called    int
}

func (m *mockModel) Name() string {
	return m.name
}

func (m *mockModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		m.called++
		if m.shouldErr {
			yield(nil, errors.New("simulated error on "+m.name))
			return
		}

		resp := &model.LLMResponse{
			Content: genai.NewContentFromParts([]*genai.Part{
				genai.NewPartFromText(m.response),
			}, genai.RoleModel),
		}
		yield(resp, nil)
	}
}

func TestFallbackModel_PrimarySucceeds(t *testing.T) {
	primary := &mockModel{name: "primary-model", shouldErr: false, response: "primary response"}
	backup := &mockModel{name: "backup-model", shouldErr: false, response: "backup response"}

	fb := NewFallbackModel(primary, backup)

	ctx := context.Background()
	req := &model.LLMRequest{}

	var outputs []string
	for resp, err := range fb.GenerateContent(ctx, req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != nil {
			for _, p := range resp.Content.Parts {
				outputs = append(outputs, p.Text)
			}
		}
	}

	if primary.called != 1 {
		t.Errorf("expected primary to be called once, got %d", primary.called)
	}
	if backup.called != 0 {
		t.Errorf("expected backup to NOT be called, got %d", backup.called)
	}
	if len(outputs) != 1 || outputs[0] != "primary response" {
		t.Errorf("unexpected output: %v", outputs)
	}
}

func TestFallbackModel_PrimaryFails_BackupSucceeds(t *testing.T) {
	primary := &mockModel{name: "primary-model", shouldErr: true}
	backup := &mockModel{name: "backup-model", shouldErr: false, response: "backup success"}

	fb := NewFallbackModel(primary, backup)

	ctx := context.Background()
	req := &model.LLMRequest{}

	var outputs []string
	for resp, err := range fb.GenerateContent(ctx, req, false) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Content != nil {
			for _, p := range resp.Content.Parts {
				outputs = append(outputs, p.Text)
			}
		}
	}

	if primary.called != 1 {
		t.Errorf("expected primary to be called once, got %d", primary.called)
	}
	if backup.called != 1 {
		t.Errorf("expected backup to be called once, got %d", backup.called)
	}
	if len(outputs) != 1 || outputs[0] != "backup success" {
		t.Errorf("unexpected output: %v", outputs)
	}
}

func TestFallbackModel_AllFail(t *testing.T) {
	primary := &mockModel{name: "primary-model", shouldErr: true}
	backup := &mockModel{name: "backup-model", shouldErr: true}

	fb := NewFallbackModel(primary, backup)

	ctx := context.Background()
	req := &model.LLMRequest{}

	var gotErr error
	for _, err := range fb.GenerateContent(ctx, req, false) {
		if err != nil {
			gotErr = err
		}
	}

	if gotErr == nil {
		t.Fatalf("expected error when all models fail, got nil")
	}
	if !strings.Contains(gotErr.Error(), "all models in fallback chain failed") {
		t.Errorf("unexpected error message: %v", gotErr)
	}
}

func TestFallbackModel_Name(t *testing.T) {
	m1 := &mockModel{name: "m1"}
	m2 := &mockModel{name: "m2"}

	fb := NewFallbackModel(m1, m2)
	expectedName := "fallback[m1 -> m2]"
	if fb.Name() != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, fb.Name())
	}
}
