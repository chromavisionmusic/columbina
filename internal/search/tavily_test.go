package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient_Options(t *testing.T) {
	customHTTP := &http.Client{Timeout: 5 * time.Second}
	customCache := NewCache(10 * time.Minute)
	defer customCache.Close()

	client := NewClient("test-key",
		WithBaseURL("https://custom.tavily.com"),
		WithHTTPClient(customHTTP),
		WithMaxResults(8),
		WithCache(customCache),
	)
	defer client.Close()

	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", client.apiKey, "test-key")
	}
	if client.baseURL != "https://custom.tavily.com" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://custom.tavily.com")
	}
	if client.httpClient != customHTTP {
		t.Error("custom HTTP client not set")
	}
	if client.maxResults != 8 {
		t.Errorf("maxResults = %d, want 8", client.maxResults)
	}
	if client.Cache() != customCache {
		t.Error("custom cache not set")
	}

	// Test WithTTL option
	clientWithTTL := NewClient("key", WithTTL(3*time.Minute))
	defer clientWithTTL.Close()
	if clientWithTTL.cache.ttl != 3*time.Minute {
		t.Errorf("ttl = %v, want 3m", clientWithTTL.cache.ttl)
	}
}

func TestClient_Search_EmptyQuery(t *testing.T) {
	client := NewClient("test-key")
	defer client.Close()

	res, err := client.Search(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error for empty query: %v", err)
	}
	if res.Query != "" {
		t.Errorf("expected empty query result, got %q", res.Query)
	}
}

func TestClient_Search_MissingAPIKey(t *testing.T) {
	client := NewClient("")
	defer client.Close()

	_, err := client.Search(context.Background(), "test query")
	if err == nil || !strings.Contains(err.Error(), "tavily API key is missing") {
		t.Fatalf("expected missing API key error, got %v", err)
	}
}

func TestClient_Search_SuccessAndCaching(t *testing.T) {
	var requestCount int32
	var receivedPayload tavilySearchRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer my-secret-key" {
			t.Errorf("Authorization header = %q, want 'Bearer my-secret-key'", authHeader)
		}

		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)

		resp := tavilySearchResponse{
			Query:  receivedPayload.Query,
			Answer: "Columbina is the Third Fatui Harbinger.",
			Results: []tavilyResultObject{
				{
					Title:   "Columbina Info",
					URL:     "https://genshin.example.com",
					Content: "Details about Damselette.",
					Score:   0.98,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient("Bearer my-secret-key",
		WithBaseURL(server.URL),
		WithMaxResults(3),
	)
	defer client.Close()

	ctx := context.Background()
	res, err := client.Search(ctx, "who is columbina")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Fatalf("expected 1 HTTP request, got %d", requestCount)
	}
	if receivedPayload.Query != "who is columbina" {
		t.Errorf("payload Query = %q", receivedPayload.Query)
	}
	if receivedPayload.MaxResults != 3 {
		t.Errorf("payload MaxResults = %d, want 3", receivedPayload.MaxResults)
	}
	if res.Answer != "Columbina is the Third Fatui Harbinger." {
		t.Errorf("Answer = %q", res.Answer)
	}
	if len(res.Results) != 1 || res.Results[0].Title != "Columbina Info" {
		t.Errorf("Results mismatch: %+v", res.Results)
	}

	// Second query should hit cache without calling server
	res2, err := client.Search(ctx, "who is columbina")
	if err != nil {
		t.Fatalf("cached Search failed: %v", err)
	}
	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected still 1 HTTP request due to cache, got %d", requestCount)
	}
	if res2.Answer != res.Answer {
		t.Errorf("cached answer mismatch: %q", res2.Answer)
	}
}

func TestClient_Search_TruncatesLongQuery(t *testing.T) {
	var requestedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req tavilySearchRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		requestedQuery = req.Query

		_ = json.NewEncoder(w).Encode(tavilySearchResponse{
			Query: requestedQuery,
		})
	}))
	defer server.Close()

	client := NewClient("test-key", WithBaseURL(server.URL))
	defer client.Close()

	longQuery := strings.Repeat("a", 500)
	_, err := client.Search(context.Background(), longQuery)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(requestedQuery) != 400 {
		t.Errorf("expected truncated query len 400, got %d", len(requestedQuery))
	}
}

func TestClient_Search_ErrorResponses(t *testing.T) {
	t.Run("HTTP 500 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient("test-key", WithBaseURL(server.URL))
		defer client.Close()

		_, err := client.Search(context.Background(), "query")
		if err == nil || !strings.Contains(err.Error(), "tavily returned HTTP 500") {
			t.Fatalf("expected HTTP 500 error, got %v", err)
		}
	})

	t.Run("API error field in response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(tavilySearchResponse{
				Error: "rate limit exceeded",
			})
		}))
		defer server.Close()

		client := NewClient("test-key", WithBaseURL(server.URL))
		defer client.Close()

		_, err := client.Search(context.Background(), "query")
		if err == nil || !strings.Contains(err.Error(), "rate limit exceeded") {
			t.Fatalf("expected rate limit error, got %v", err)
		}
	})

	t.Run("API detail string error in response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(tavilySearchResponse{
				Detail: "invalid credentials",
			})
		}))
		defer server.Close()

		client := NewClient("test-key", WithBaseURL(server.URL))
		defer client.Close()

		_, err := client.Search(context.Background(), "query")
		if err == nil || !strings.Contains(err.Error(), "invalid credentials") {
			t.Fatalf("expected invalid credentials error, got %v", err)
		}
	})

	t.Run("invalid json response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("{invalid-json"))
		}))
		defer server.Close()

		client := NewClient("test-key", WithBaseURL(server.URL))
		defer client.Close()

		_, err := client.Search(context.Background(), "query")
		if err == nil || !strings.Contains(err.Error(), "failed to decode response") {
			t.Fatalf("expected decode error, got %v", err)
		}
	})
}

func TestClient_SearchToolHandler(t *testing.T) {
	t.Run("empty query argument", func(t *testing.T) {
		client := NewClient("test-key")
		defer client.Close()

		res, err := client.SearchToolHandler(nil, SearchArgs{Query: "   "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Answer != "No search query provided." {
			t.Errorf("Answer = %q, want 'No search query provided.'", res.Answer)
		}
	})

	t.Run("silent fallback on API error", func(t *testing.T) {
		// Server returning 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "down", http.StatusInternalServerError)
		}))
		defer server.Close()

		client := NewClient("test-key", WithBaseURL(server.URL))
		defer client.Close()

		res, err := client.SearchToolHandler(nil, SearchArgs{Query: "test"})
		if err != nil {
			t.Fatalf("SearchToolHandler should not return error on failure, got %v", err)
		}
		if res.Answer != "No external search results found." {
			t.Errorf("Answer = %q, want 'No external search results found.'", res.Answer)
		}
	})

	t.Run("empty results fallback message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(tavilySearchResponse{
				Query:   "nothing",
				Answer:  "",
				Results: nil,
			})
		}))
		defer server.Close()

		client := NewClient("test-key", WithBaseURL(server.URL))
		defer client.Close()

		res, err := client.SearchToolHandler(nil, SearchArgs{Query: "nothing"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Answer != "No external search results found." {
			t.Errorf("Answer = %q, want 'No external search results found.'", res.Answer)
		}
	})
}

func TestClient_AsTool(t *testing.T) {
	client := NewClient("test-key")
	defer client.Close()

	tl, err := client.AsTool()
	if err != nil {
		t.Fatalf("AsTool() unexpected error: %v", err)
	}
	if tl == nil {
		t.Fatal("expected non-nil Tool")
	}
	if tl.Name() != "web_search" {
		t.Errorf("Tool name = %q, want 'web_search'", tl.Name())
	}
}
