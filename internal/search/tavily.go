package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"charm.land/log/v2"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
)

const (
	// DefaultTavilyBaseURL points to the Tavily search endpoint.
	DefaultTavilyBaseURL = "https://api.tavily.com/search"

	// DefaultSearchDepth uses basic search depth for faster responses.
	DefaultSearchDepth = "basic"

	// DefaultMaxResults fetches the top 5 results (3-5 results requirement).
	DefaultMaxResults = 5

	// DefaultCacheTTL keeps query responses cached in memory for 20 minutes.
	DefaultCacheTTL = 20 * time.Minute

	// DefaultHTTPTimeout limits outgoing HTTP requests to Tavily.
	DefaultHTTPTimeout = 10 * time.Second
)

// SearchArgs defines the arguments received by the web search function tool.
type SearchArgs struct {
	Query string `json:"query" jsonschema:"The search query string to look up on the web"`
}

// SearchResultItem represents an individual search snippet returned from Tavily.
type SearchResultItem struct {
	Title   string  `json:"title"`
	URL     string  `json:"url,omitempty"`
	Content string  `json:"content"`
	Score   float64 `json:"score,omitempty"`
}

// SearchResult represents the full aggregated search output returned to the LLM agent.
type SearchResult struct {
	Query   string             `json:"query"`
	Answer  string             `json:"answer,omitempty"`
	Results []SearchResultItem `json:"results"`
}

type tavilySearchRequest struct {
	APIKey        string `json:"api_key"`
	Query         string `json:"query"`
	SearchDepth   string `json:"search_depth"`
	MaxResults    int    `json:"max_results"`
	IncludeAnswer bool   `json:"include_answer"`
}

type tavilyResultObject struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	RawContent *string `json:"raw_content,omitempty"`
}

type tavilySearchResponse struct {
	Query        string               `json:"query"`
	Answer       string               `json:"answer"`
	Results      []tavilyResultObject `json:"results"`
	ResponseTime float64              `json:"response_time"`
	Detail       any                  `json:"detail,omitempty"`
	Error        string               `json:"error,omitempty"`
}

// Client executes searches against the Tavily API with caching and error handling.
type Client struct {
	apiKey        string
	baseURL       string
	searchDepth   string
	maxResults    int
	includeAnswer bool
	httpClient    *http.Client
	cache         *Cache
}

// Option configures Client behavior.
type Option func(*Client)

// WithBaseURL overrides the search API endpoint URL (useful for testing).
func WithBaseURL(url string) Option {
	return func(c *Client) {
		if strings.TrimSpace(url) != "" {
			c.baseURL = url
		}
	}
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithTTL sets a custom cache time-to-live.
func WithTTL(ttl time.Duration) Option {
	return func(c *Client) {
		if c.cache != nil {
			c.cache.Close()
		}
		c.cache = NewCache(ttl)
	}
}

// WithMaxResults sets the maximum number of results to fetch.
func WithMaxResults(n int) Option {
	return func(c *Client) {
		if n > 0 {
			c.maxResults = n
		}
	}
}

// WithCache sets an existing cache instance.
func WithCache(cache *Cache) Option {
	return func(c *Client) {
		if cache != nil {
			if c.cache != nil {
				c.cache.Close()
			}
			c.cache = cache
		}
	}
}

// NewClient initializes a new Tavily search client.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:        strings.TrimSpace(apiKey),
		baseURL:       DefaultTavilyBaseURL,
		searchDepth:   DefaultSearchDepth,
		maxResults:    DefaultMaxResults,
		includeAnswer: true,
		httpClient:    &http.Client{Timeout: DefaultHTTPTimeout},
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.cache == nil {
		c.cache = NewCache(DefaultCacheTTL)
	}

	return c
}

// Cache returns the client's in-memory TTL cache.
func (c *Client) Cache() *Cache {
	return c.cache
}

// Close closes background resources such as the cache cleaner.
func (c *Client) Close() {
	if c.cache != nil {
		c.cache.Close()
	}
}

// Search queries Tavily for live web results, consulting and updating the TTL cache.
func (c *Client) Search(ctx context.Context, query string) (SearchResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return SearchResult{Query: ""}, nil
	}

	// Guard against excessively long queries triggering upstream gateway rejection
	if len(query) > 400 {
		query = strings.TrimSpace(query[:400])
	}

	// 1. Check TTL cache
	if c.cache != nil {
		if cached, hit := c.cache.Get(query); hit {
			log.Debugf("Tavily search cache hit for: %q", query)
			return cached, nil
		}
	}

	if c.apiKey == "" {
		return SearchResult{Query: query}, fmt.Errorf("tavily API key is missing")
	}

	// 2. Prepare HTTP payload and authentication
	cleanKey := strings.TrimPrefix(c.apiKey, "Bearer ")
	reqPayload := tavilySearchRequest{
		APIKey:        cleanKey,
		Query:         query,
		SearchDepth:   c.searchDepth,
		MaxResults:    c.maxResults,
		IncludeAnswer: c.includeAnswer,
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return SearchResult{Query: query}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return SearchResult{Query: query}, fmt.Errorf("failed to build http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cleanKey)

	// 3. Execute HTTP request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return SearchResult{Query: query}, fmt.Errorf("tavily request failed: %w", err)
	}
	defer httpResp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return SearchResult{Query: query}, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return SearchResult{Query: query}, fmt.Errorf("tavily returned HTTP %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	var tavilyResp tavilySearchResponse
	if err := json.Unmarshal(bodyBytes, &tavilyResp); err != nil {
		return SearchResult{Query: query}, fmt.Errorf("failed to decode response: %w", err)
	}

	if tavilyResp.Error != "" {
		return SearchResult{Query: query}, fmt.Errorf("tavily api error: %s", tavilyResp.Error)
	}
	if detailStr, ok := tavilyResp.Detail.(string); ok && detailStr != "" {
		return SearchResult{Query: query}, fmt.Errorf("tavily api error: %s", detailStr)
	}

	result := SearchResult{
		Query:   query,
		Answer:  strings.TrimSpace(tavilyResp.Answer),
		Results: make([]SearchResultItem, 0, len(tavilyResp.Results)),
	}

	for _, item := range tavilyResp.Results {
		result.Results = append(result.Results, SearchResultItem{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.URL),
			Content: strings.TrimSpace(item.Content),
			Score:   item.Score,
		})
	}

	// 4. Update cache
	if c.cache != nil {
		c.cache.Set(query, result)
	}

	return result, nil
}

// SearchToolHandler implements the ADK functiontool handler with graceful silent fallback.
// If Tavily times out, rate limits, or errors, it logs a warning and returns an empty/fallback
// result to the agent so Columbina continues with her internal knowledge without crashing.
func (c *Client) SearchToolHandler(ctx agent.Context, args SearchArgs) (SearchResult, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return SearchResult{
			Query:   "",
			Answer:  "No search query provided.",
			Results: []SearchResultItem{},
		}, nil
	}

	res, err := c.Search(ctx, query)
	if err != nil {
		log.Warnf("Tavily web search error for query %q: %v; silently falling back", query, err)
		return SearchResult{
			Query:   query,
			Answer:  "No external search results found.",
			Results: []SearchResultItem{},
		}, nil
	}

	if strings.TrimSpace(res.Answer) == "" && len(res.Results) == 0 {
		res.Answer = "No external search results found."
	}

	return res, nil
}

// AsTool converts the Tavily client into an ADK Go v2 function tool.
func (c *Client) AsTool() (tool.Tool, error) {
	return functiontool.New(functiontool.Config{
		Name:        "web_search",
		Description: "Search the web for up-to-date information, current news, recent events, weather, facts, and live data. Use only when external current information is needed.",
	}, c.SearchToolHandler)
}
