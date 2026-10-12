package ai

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"charm.land/log/v2"
	"google.golang.org/adk/v2/model"
)

// FallbackModel wraps multiple model.LLM instances and tries them sequentially.
// If a model encounters an error before yielding any response, execution automatically
// fails over to the next candidate model.
type FallbackModel struct {
	models []model.LLM
}

// NewFallbackModel returns a model.LLM with sequential failover capabilities.
func NewFallbackModel(models ...model.LLM) model.LLM {
	if len(models) == 0 {
		panic("at least one model is required")
	}
	if len(models) == 1 {
		return models[0]
	}
	return &FallbackModel{
		models: models,
	}
}

// Name returns the composite name of the fallback chain.
func (f *FallbackModel) Name() string {
	names := make([]string, len(f.models))
	for i, m := range f.models {
		names[i] = m.Name()
	}
	return fmt.Sprintf("fallback[%s]", strings.Join(names, " -> "))
}

// GenerateContent attempts generation across each model in order until one succeeds.
func (f *FallbackModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		var lastErr error

		for i, m := range f.models {
			hasYielded := false
			var modelErr error

			var candidateReq model.LLMRequest
			if req != nil {
				candidateReq = *req
				candidateReq.Model = m.Name()
			}

			for resp, err := range m.GenerateContent(ctx, &candidateReq, stream) {
				if err != nil {
					modelErr = err
					break
				}
				hasYielded = true
				if !yield(resp, nil) {
					return // Caller broke loop early
				}
			}

			// If completed successfully and produced output, finish.
			if modelErr == nil && hasYielded {
				return
			}

			if modelErr != nil {
				lastErr = modelErr
				log.Warnf("Model %s failed: %v", m.Name(), modelErr)

				// If the model already started yielding tokens to the caller before failing,
				// we cannot seamlessly failover mid-stream without corrupting the output.
				if hasYielded {
					yield(nil, modelErr)
					return
				}

				if i < len(f.models)-1 {
					log.Infof("Failing over to next model: %s", f.models[i+1].Name())
				}
			}
		}

		if lastErr != nil {
			yield(nil, fmt.Errorf("all models in fallback chain failed: %w", lastErr))
		}
	}
}
