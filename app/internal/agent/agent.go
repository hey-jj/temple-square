// Package agent implements a parallel agent architecture using direct Gemini REST API calls.
// Architecture: Orchestrator (1 LLM call) → Parallel Search Agents (each 1 tool call + 1 format call)
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
)

// PresidentsOrchestratorResponse is the structured output for presidents keywords.
type PresidentsOrchestratorResponse struct {
	Safe     bool   `json:"safe"`
	Reason   string `json:"reason,omitempty"`
	Keywords struct {
		PresidentsOaks    string `json:"presidents_oaks"`
		PresidentsGeneral string `json:"presidents_general"`
	} `json:"keywords"`
}

// LeadersOrchestratorResponse is the structured output for leaders keywords.
type LeadersOrchestratorResponse struct {
	Keywords struct {
		LeadersFirstPres string `json:"leaders_first_presidency"`
		LeadersQ12       string `json:"leaders_q12"`
		LeadersOther     string `json:"leaders_other"`
	} `json:"keywords"`
}

// ScripturesOrchestratorResponse is the structured output for scripture keywords.
type ScripturesOrchestratorResponse struct {
	Keywords struct {
		ScripturesBible string `json:"scriptures_bible"`
		ScripturesBoM   string `json:"scriptures_bom"`
		ScripturesOther string `json:"scriptures_other"`
	} `json:"keywords"`
}

// StructuredQuote defines the schema for a quote response
type StructuredQuote struct {
	Speaker    string `json:"speaker"`
	Title      string `json:"title"`
	Conference string `json:"conference"`
	Quote      string `json:"quote"`
	Headshot   string `json:"headshot,omitempty"`
}

// StructuredScripture defines the schema for a scripture response
type StructuredScripture struct {
	Volume      string            `json:"volume"`
	Reference   string            `json:"reference"`
	Text        string            `json:"text"`
	RelatedTalk *RelatedTalkQuote `json:"related_talk,omitempty"`
}

// RelatedTalkQuote is a smaller quote from a talk referencing the scripture
type RelatedTalkQuote struct {
	Speaker string `json:"speaker"`
	Title   string `json:"title"`
	Quote   string `json:"quote"`
}

// PresidentsResponse is the structured output for presidents agent
type PresidentsResponse struct {
	Quotes []StructuredQuote `json:"quotes"`
}

// LeadersResponse is the structured output for leaders agent
type LeadersResponse struct {
	Quotes []StructuredQuote `json:"quotes"`
}

// ScripturesResponse is the structured output for scriptures agent
type ScripturesResponse struct {
	Scriptures []StructuredScripture `json:"scriptures"`
}

// Config holds agent configuration
type Config struct {
	ToolboxURL string
	APIKey     string
}

// ProphetAgent is the main agent that coordinates parallel sub-agents
type ProphetAgent struct {
	client     *GeminiClient
	toolboxURL string

	initOnce      sync.Once
	initErr       error
	toolboxClient *core.ToolboxClient
	allTools      map[string]*core.ToolboxTool
}

// AgentResult contains the result from a single sub-agent
type AgentResult struct {
	AgentName string
	Content   string
	Error     error
}

// New creates a new prophet agent
func New(ctx context.Context, cfg Config) (*ProphetAgent, error) {
	client, err := NewGeminiClient(cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	toolboxURL := cfg.ToolboxURL
	if toolboxURL == "" {
		toolboxURL = os.Getenv("TOOLBOX_URL")
		if toolboxURL == "" {
			toolboxURL = "http://127.0.0.1:5000"
		}
	}

	log.Printf("Prophet agent created (tools loaded on first request)")

	return &ProphetAgent{
		client:     client,
		toolboxURL: toolboxURL,
	}, nil
}

// ensureInitialized loads MCP Toolbox tools on first request
func (a *ProphetAgent) ensureInitialized(ctx context.Context) error {
	a.initOnce.Do(func() {
		log.Printf("Loading MCP Toolbox tools...")

		// Configure HTTP client with proper connection pooling for parallel requests
		httpClient := &http.Client{
			Timeout: 120 * time.Second, // Allow higher parallelism without client timeouts
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				MaxConnsPerHost:     100,
				IdleConnTimeout:     90 * time.Second,
			},
		}

		toolboxClient, err := core.NewToolboxClient(a.toolboxURL, core.WithHTTPClient(httpClient))
		if err != nil {
			a.initErr = fmt.Errorf("failed to create MCP Toolbox client: %w", err)
			return
		}
		a.toolboxClient = toolboxClient
		a.allTools = make(map[string]*core.ToolboxTool)

		// Load all toolsets and flatten into a single map
		for _, toolset := range []string{"presidents", "leaders", "scriptures"} {
			tools, err := toolboxClient.LoadToolset(toolset, ctx)
			if err != nil {
				a.initErr = fmt.Errorf("failed to load %s toolset: %w", toolset, err)
				return
			}
			for _, t := range tools {
				a.allTools[t.Name()] = t
			}
		}

		log.Printf("Loaded %d tools total", len(a.allTools))
	})
	return a.initErr
}

// Run executes orchestrator then parallel search agents
func (a *ProphetAgent) Run(ctx context.Context, question string) <-chan AgentResult {
	results := make(chan AgentResult, 32) // buffered for cascade fan-out

	go func() {
		defer close(results)

		// Ensure tools are loaded
		if err := a.ensureInitialized(ctx); err != nil {
			results <- AgentResult{Error: err}
			return
		}

		// STEP 1: Presidents orchestrator (safety + keywords)
		log.Printf("[orchestrator-presidents] Starting with question: %s", question)
		presOrch, err := a.runOrchestratorPresidents(ctx, question)
		if err != nil {
			results <- AgentResult{Error: fmt.Errorf("presidents orchestrator failed: %w", err)}
			return
		}

		// Check safety
		if !presOrch.Safe {
			log.Printf("[orchestrator-presidents] Blocked unsafe content: %s", presOrch.Reason)
			results <- AgentResult{
				AgentName: "orchestrator",
				Error:     fmt.Errorf("blocked: %s", presOrch.Reason),
			}
			return
		}

		// STEP 1b: Leaders + Scriptures orchestrators (parallel)
		var leadersOrch *LeadersOrchestratorResponse
		var scripturesOrch *ScripturesOrchestratorResponse
		var leadersErr error
		var scripturesErr error

		leadersReady := make(chan struct{})
		scripturesReady := make(chan struct{})

		go func() {
			leadersOrch, leadersErr = a.runOrchestratorLeaders(ctx, question)
			close(leadersReady)
		}()
		go func() {
			scripturesOrch, scripturesErr = a.runOrchestratorScriptures(ctx, question)
			close(scripturesReady)
		}()

		log.Printf("[orchestrator] Keywords generated, launching cascade")

		var leadersOnce sync.Once
		leadersDone := make(chan struct{})

		startLeaders := func() {
			leadersOnce.Do(func() {
				go func() {
					defer close(leadersDone)
					<-leadersReady
					if leadersErr != nil {
						results <- AgentResult{Error: fmt.Errorf("leaders orchestrator failed: %w", leadersErr)}
						return
					}
					if leadersOrch == nil {
						results <- AgentResult{Error: fmt.Errorf("leaders orchestrator returned no data")}
						return
					}
					var leadersWG sync.WaitGroup

					leadersWG.Add(1)
					go func() {
						defer leadersWG.Done()
						content, err := a.runSearchAgent(ctx, "leaders_eyring",
							leadersOrch.Keywords.LeadersFirstPres,
							"get_leaders_talks",
							map[string]any{"query": leadersOrch.Keywords.LeadersFirstPres, "limit": 3},
							leadersEyringPrompt, quotesSchema)
						results <- AgentResult{AgentName: "leaders_agent", Content: content, Error: err}
					}()

					leadersWG.Add(1)
					go func() {
						defer leadersWG.Done()
						content, err := a.runSearchAgent(ctx, "leaders_christofferson",
							leadersOrch.Keywords.LeadersFirstPres,
							"get_leaders_talks",
							map[string]any{"query": leadersOrch.Keywords.LeadersFirstPres, "limit": 3},
							leadersChristoffersonPrompt, quotesSchema)
						results <- AgentResult{AgentName: "leaders_agent", Content: content, Error: err}
					}()

					leadersWG.Add(1)
					go func() {
						defer leadersWG.Done()
						content, err := a.runSearchAgent(ctx, "leaders_q12_a",
							leadersOrch.Keywords.LeadersQ12,
							"get_leaders_talks",
							map[string]any{"query": leadersOrch.Keywords.LeadersQ12, "limit": 3},
							leadersQ12PromptA, quotesSchema)
						results <- AgentResult{AgentName: "leaders_agent", Content: content, Error: err}
					}()

					leadersWG.Add(1)
					go func() {
						defer leadersWG.Done()
						content, err := a.runSearchAgent(ctx, "leaders_q12_b",
							leadersOrch.Keywords.LeadersQ12,
							"get_leaders_talks",
							map[string]any{"query": leadersOrch.Keywords.LeadersQ12, "limit": 3},
							leadersQ12PromptB, quotesSchema)
						results <- AgentResult{AgentName: "leaders_agent", Content: content, Error: err}
					}()

					leadersWG.Add(1)
					go func() {
						defer leadersWG.Done()
						content, err := a.runSearchAgent(ctx, "leaders_other_a",
							leadersOrch.Keywords.LeadersOther,
							"search_talks",
							map[string]any{"query": leadersOrch.Keywords.LeadersOther, "limit": 3},
							leadersOtherPromptA, quotesSchema)
						results <- AgentResult{AgentName: "leaders_agent", Content: content, Error: err}
					}()

					leadersWG.Add(1)
					go func() {
						defer leadersWG.Done()
						content, err := a.runSearchAgent(ctx, "leaders_other_b",
							leadersOrch.Keywords.LeadersOther,
							"search_talks",
							map[string]any{"query": leadersOrch.Keywords.LeadersOther, "limit": 3},
							leadersOtherPromptB, quotesSchema)
						results <- AgentResult{AgentName: "leaders_agent", Content: content, Error: err}
					}()

					leadersWG.Wait()
				}()
			})
		}

		// STEP 2: Presidents section (start immediately)
		var presidentsWG sync.WaitGroup

		presidentsWG.Add(1)
		go func() {
			defer presidentsWG.Done()
			content, err := a.runSearchAgent(ctx, "presidents_oaks",
				presOrch.Keywords.PresidentsOaks,
				"search_talks_by_speaker",
				map[string]any{"speaker_slug": "dallin-oaks", "limit": 3},
				presidentsOaksPrompt, quotesSchema)
			results <- AgentResult{AgentName: "presidents_agent", Content: content, Error: err}
			startLeaders()
		}()

		presidentsWG.Add(1)
		go func() {
			defer presidentsWG.Done()
			content, err := a.runSearchAgent(ctx, "presidents_nelson",
				presOrch.Keywords.PresidentsGeneral,
				"search_talks_by_speaker",
				map[string]any{"speaker_slug": "russell-nelson", "limit": 3},
				presidentsNelsonPrompt, quotesSchema)
			results <- AgentResult{AgentName: "presidents_agent", Content: content, Error: err}
			startLeaders()
		}()

		presidentsWG.Add(1)
		go func() {
			defer presidentsWG.Done()
			content, err := a.runSearchAgent(ctx, "presidents_general",
				presOrch.Keywords.PresidentsGeneral,
				"get_presidents_talks",
				map[string]any{"query": presOrch.Keywords.PresidentsGeneral, "limit": 3},
				presidentsGeneralPrompt, quotesSchema)
			results <- AgentResult{AgentName: "presidents_agent", Content: content, Error: err}
			startLeaders()
		}()

		presidentsWG.Wait()
		startLeaders()
		<-leadersDone

		// STEP 3: Scriptures section (start after leaders complete)
		log.Printf("[scriptures] Starting for question: %s", question)
		<-scripturesReady
		if scripturesErr != nil {
			results <- AgentResult{Error: fmt.Errorf("scriptures orchestrator failed: %w", scripturesErr)}
			return
		}
		if scripturesOrch == nil {
			results <- AgentResult{Error: fmt.Errorf("scriptures orchestrator returned no data")}
			return
		}

		var scripturesWG sync.WaitGroup

		// Bible: 2 cards (Old Testament + New Testament)
		scripturesWG.Add(1)
		go func() {
			defer scripturesWG.Done()
			query := fmt.Sprintf("%s Bible Old Testament New Testament", scripturesOrch.Keywords.ScripturesBible)
			content, err := a.runSearchAgent(ctx, "scriptures_bible",
				query,
				"search_scriptures",
				map[string]any{"query": query, "limit": 12},
				scripturesBiblePrompt, scripturesCategorySchema)
			results <- AgentResult{AgentName: "scriptures_bible", Content: content, Error: err}
		}()

		// Book of Mormon: 2 cards
		scripturesWG.Add(1)
		go func() {
			defer scripturesWG.Done()
			query := fmt.Sprintf("%s Book of Mormon", scripturesOrch.Keywords.ScripturesBoM)
			content, err := a.runSearchAgent(ctx, "scriptures_bom",
				query,
				"search_scriptures",
				map[string]any{"query": query, "limit": 12},
				scripturesBoMPrompt, scripturesCategorySchema)
			results <- AgentResult{AgentName: "scriptures_bom", Content: content, Error: err}
		}()

		// Other scriptures: 2 cards (D&C + Pearl of Great Price)
		scripturesWG.Add(1)
		go func() {
			defer scripturesWG.Done()
			query := fmt.Sprintf("%s Doctrine and Covenants Pearl of Great Price", scripturesOrch.Keywords.ScripturesOther)
			content, err := a.runSearchAgent(ctx, "scriptures_other",
				query,
				"search_scriptures",
				map[string]any{"query": query, "limit": 12},
				scripturesOtherPrompt, scripturesCategorySchema)
			results <- AgentResult{AgentName: "scriptures_other", Content: content, Error: err}
		}()

		scripturesWG.Wait()
	}()

	return results
}

// runOrchestratorPresidents generates safety + presidents keywords
func (a *ProphetAgent) runOrchestratorPresidents(ctx context.Context, question string) (*PresidentsOrchestratorResponse, error) {
	temp := float32(1.0)

	req := &GenerateRequest{
		Contents: []*Content{{
			Parts: []*Part{{Text: question}},
			Role:  "user",
		}},
		SystemInstruct: &Content{
			Parts: []*Part{{Text: orchestratorPresidentsPrompt}},
			Role:  "system",
		},
		GenerationConfig: &GenerationConfig{
			Temperature:        &temp,
			MaxOutputTokens:    64000,
			ResponseMIMEType:   "application/json",
			ResponseJSONSchema: orchestratorPresidentsSchema,
			ThinkingConfig:     &ThinkingConfig{ThinkingLevel: "high"},
		},
		SafetySettings: DefaultSafetySettings(),
	}

	resp, err := a.client.GenerateContent(ctx, req)
	if err != nil {
		return nil, err
	}

	text := resp.ExtractText()
	log.Printf("[orchestrator-presidents] Response: %s", text)

	var orchResp PresidentsOrchestratorResponse
	if err := json.Unmarshal([]byte(text), &orchResp); err != nil {
		return nil, fmt.Errorf("failed to parse presidents orchestrator response: %w", err)
	}

	return &orchResp, nil
}

// runOrchestratorLeaders generates leaders keywords
func (a *ProphetAgent) runOrchestratorLeaders(ctx context.Context, question string) (*LeadersOrchestratorResponse, error) {
	temp := float32(1.0)

	req := &GenerateRequest{
		Contents: []*Content{{
			Parts: []*Part{{Text: question}},
			Role:  "user",
		}},
		SystemInstruct: &Content{
			Parts: []*Part{{Text: orchestratorLeadersPrompt}},
			Role:  "system",
		},
		GenerationConfig: &GenerationConfig{
			Temperature:        &temp,
			MaxOutputTokens:    64000,
			ResponseMIMEType:   "application/json",
			ResponseJSONSchema: orchestratorLeadersSchema,
			ThinkingConfig:     &ThinkingConfig{ThinkingLevel: "high"},
		},
		SafetySettings: DefaultSafetySettings(),
	}

	resp, err := a.client.GenerateContent(ctx, req)
	if err != nil {
		return nil, err
	}

	text := resp.ExtractText()
	log.Printf("[orchestrator-leaders] Response: %s", text)

	var orchResp LeadersOrchestratorResponse
	if err := json.Unmarshal([]byte(text), &orchResp); err != nil {
		return nil, fmt.Errorf("failed to parse leaders orchestrator response: %w", err)
	}

	return &orchResp, nil
}

// runOrchestratorScriptures generates scripture keywords
func (a *ProphetAgent) runOrchestratorScriptures(ctx context.Context, question string) (*ScripturesOrchestratorResponse, error) {
	temp := float32(1.0)

	req := &GenerateRequest{
		Contents: []*Content{{
			Parts: []*Part{{Text: question}},
			Role:  "user",
		}},
		SystemInstruct: &Content{
			Parts: []*Part{{Text: orchestratorScripturesPrompt}},
			Role:  "system",
		},
		GenerationConfig: &GenerationConfig{
			Temperature:        &temp,
			MaxOutputTokens:    64000,
			ResponseMIMEType:   "application/json",
			ResponseJSONSchema: orchestratorScripturesSchema,
			ThinkingConfig:     &ThinkingConfig{ThinkingLevel: "high"},
		},
		SafetySettings: DefaultSafetySettings(),
	}

	resp, err := a.client.GenerateContent(ctx, req)
	if err != nil {
		return nil, err
	}

	text := resp.ExtractText()
	log.Printf("[orchestrator-scriptures] Response: %s", text)

	var orchResp ScripturesOrchestratorResponse
	if err := json.Unmarshal([]byte(text), &orchResp); err != nil {
		return nil, fmt.Errorf("failed to parse scriptures orchestrator response: %w", err)
	}

	return &orchResp, nil
}

// runSearchAgent executes a single search and formats results
func (a *ProphetAgent) runSearchAgent(ctx context.Context, name, keywords, toolName string, toolArgs map[string]any, formatPrompt string, schema map[string]any) (string, error) {
	start := time.Now()
	log.Printf("[%s] Starting - keywords: %s", name, keywords)

	// Get the tool
	tool, ok := a.allTools[toolName]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", toolName)
	}

	// Execute the search (ONE tool call)
	toolStart := time.Now()
	result, err := tool.Invoke(ctx, toolArgs)
	toolDuration := time.Since(toolStart)
	if err != nil {
		log.Printf("[%s] Tool failed after %v: %v", name, toolDuration, err)
		return "", fmt.Errorf("tool %s failed: %w", toolName, err)
	}
	log.Printf("[%s] Tool completed in %v", name, toolDuration)

	// Convert result to JSON string for the prompt
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	log.Printf("[%s] Got %d bytes of results, starting format...", name, len(resultJSON))

	// Format results using LLM with structured output
	formatStart := time.Now()
	temp := float32(1.0)
	thinkingLevel := "low"
	if strings.HasPrefix(name, "scriptures_") {
		thinkingLevel = "minimal"
	}

	maxAttempts := 1
	if name == "presidents_general" {
		maxAttempts = 3
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		formatReq := &GenerateRequest{
			Contents: []*Content{{
				Parts: []*Part{{Text: fmt.Sprintf("Search results:\n%s\n\nKeywords: %s", string(resultJSON), keywords)}},
				Role:  "user",
			}},
			SystemInstruct: &Content{
				Parts: []*Part{{Text: formatPrompt}},
				Role:  "system",
			},
			GenerationConfig: &GenerationConfig{
				Temperature:        &temp,
				MaxOutputTokens:    64000,
				ResponseMIMEType:   "application/json",
				ResponseJSONSchema: schema,
				ThinkingConfig:     &ThinkingConfig{ThinkingLevel: thinkingLevel},
			},
			SafetySettings: DefaultSafetySettings(),
		}

		formatResp, err := a.client.GenerateContent(ctx, formatReq)
		formatDuration := time.Since(formatStart)
		if err != nil {
			lastErr = err
			log.Printf("[%s] Format failed after %v (attempt %d/%d): %v", name, formatDuration, attempt, maxAttempts, err)
			continue
		}

		text := formatResp.ExtractText()
		finishReason := formatResp.GetFinishReason()
		totalDuration := time.Since(start)
		log.Printf("[%s] Complete in %v (tool: %v, format: %v) - FinishReason: %s, ResponseLen: %d",
			name, totalDuration, toolDuration, formatDuration, finishReason, len(text))
		if finishReason != "STOP" && finishReason != "" {
			log.Printf("[%s] WARNING: Non-STOP finish reason, full response: %s", name, text)
		}

		if name == "presidents_general" && (finishReason == "RECITATION" || len(strings.TrimSpace(text)) == 0) {
			lastErr = fmt.Errorf("empty or recitation output")
			log.Printf("[%s] Retrying format due to %s (attempt %d/%d)", name, finishReason, attempt, maxAttempts)
			continue
		}

		return text, nil
	}

	if lastErr != nil {
		return "", fmt.Errorf("format failed: %w", lastErr)
	}
	return "", fmt.Errorf("format failed: unknown error")
}

// GenerateSummary produces a 2-3 paragraph summary from selected outputs.
func (a *ProphetAgent) GenerateSummary(ctx context.Context, question string, presidents []StructuredQuote, leaders []StructuredQuote, scriptures []StructuredScripture) (string, error) {
	temp := float32(1.0)

	payload := map[string]any{
		"question":   question,
		"presidents": limitQuotes(presidents, 3),
		"leaders":    limitQuotes(leaders, 3),
		"scriptures": limitScriptures(scriptures, 6),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary payload: %w", err)
	}

	req := &GenerateRequest{
		Contents: []*Content{{
			Parts: []*Part{{Text: string(payloadJSON)}},
			Role:  "user",
		}},
		SystemInstruct: &Content{
			Parts: []*Part{{Text: summaryPrompt}},
			Role:  "system",
		},
		GenerationConfig: &GenerationConfig{
			Temperature:        &temp,
			MaxOutputTokens:    64000,
			ResponseMIMEType:   "application/json",
			ResponseJSONSchema: summarySchema,
			ThinkingConfig:     &ThinkingConfig{ThinkingLevel: "low"},
		},
		SafetySettings: DefaultSafetySettings(),
	}

	resp, err := a.client.GenerateContent(ctx, req)
	if err != nil {
		return "", err
	}

	text := resp.ExtractText()
	log.Printf("[summary] ResponseLen: %d", len(text))
	return text, nil
}

func limitQuotes(in []StructuredQuote, n int) []StructuredQuote {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

func limitScriptures(in []StructuredScripture, n int) []StructuredScripture {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

// Schema for quote responses (presidents and leaders)
var quotesSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"quotes": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"speaker":    map[string]any{"type": "string", "minLength": 4, "maxLength": 120},
					"title":      map[string]any{"type": "string", "minLength": 4, "maxLength": 200},
					"conference": map[string]any{"type": "string", "minLength": 4, "maxLength": 80},
					"quote":      map[string]any{"type": "string", "minLength": 120, "maxLength": 2000},
					"headshot":   map[string]any{"type": "string", "maxLength": 300},
				},
				"required": []string{"speaker", "title", "conference", "quote"},
			},
		},
	},
}

// Schema for presidents orchestrator response
var orchestratorPresidentsSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"safe":   map[string]any{"type": "boolean", "description": "true if question is safe to answer"},
		"reason": map[string]any{"type": "string", "description": "reason if blocked"},
		"keywords": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"presidents_oaks":    map[string]any{"type": "string", "description": "Search keywords for Oaks talks"},
				"presidents_general": map[string]any{"type": "string", "description": "Search keywords for Nelson/Oaks talks"},
			},
			"required": []string{"presidents_oaks", "presidents_general"},
		},
	},
	"required": []string{"safe", "keywords"},
}

// Schema for leaders orchestrator response
var orchestratorLeadersSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"keywords": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"leaders_first_presidency": map[string]any{"type": "string", "description": "Search keywords for First Presidency counselors"},
				"leaders_q12":              map[string]any{"type": "string", "description": "Search keywords for Quorum of Twelve"},
				"leaders_other":            map[string]any{"type": "string", "description": "Search keywords for other leaders"},
			},
			"required": []string{"leaders_first_presidency", "leaders_q12", "leaders_other"},
		},
	},
	"required": []string{"keywords"},
}

// Schema for scriptures orchestrator response
var orchestratorScripturesSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"keywords": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"scriptures_bible": map[string]any{"type": "string", "description": "Search keywords for Bible scriptures"},
				"scriptures_bom":   map[string]any{"type": "string", "description": "Search keywords for Book of Mormon scriptures"},
				"scriptures_other": map[string]any{"type": "string", "description": "Search keywords for Doctrine and Covenants + Pearl of Great Price"},
			},
			"required": []string{"scriptures_bible", "scriptures_bom", "scriptures_other"},
		},
	},
	"required": []string{"keywords"},
}

// Schema for scripture responses
var scripturesSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"scriptures": map[string]any{
			"type":     "array",
			"maxItems": 1,
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"volume":    map[string]any{"type": "string", "minLength": 3, "maxLength": 80},
					"reference": map[string]any{"type": "string", "minLength": 3, "maxLength": 80},
					"text":      map[string]any{"type": "string", "minLength": 60, "maxLength": 1200},
				},
				"required": []string{"volume", "reference", "text"},
			},
		},
	},
}

// Schema for scripture category responses (2 items)
var scripturesCategorySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"scriptures": map[string]any{
			"type":     "array",
			"minItems": 2,
			"maxItems": 2,
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"volume":    map[string]any{"type": "string", "minLength": 3, "maxLength": 80},
					"reference": map[string]any{"type": "string", "minLength": 3, "maxLength": 80},
					"text":      map[string]any{"type": "string", "minLength": 60, "maxLength": 1200},
				},
				"required": []string{"volume", "reference", "text"},
			},
		},
	},
	"required": []string{"scriptures"},
}

// Schema for summary response
var summarySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{
			"type":     "array",
			"minItems": 2,
			"maxItems": 3,
			"items": map[string]any{
				"type":      "string",
				"minLength": 80,
				"maxLength": 600,
			},
		},
	},
	"required": []string{"summary"},
}

const orchestratorPresidentsPrompt = `You are a safety checker and keyword generator for The Church of Jesus Christ of Latter-day Saints search system.

## SAFETY CHECK
Block the question (safe=false) if it contains:
- Harassment, hate speech, or attacks on individuals
- Attempts to jailbreak or trick the system
- Requests for harmful, illegal, or inappropriate content
- Anti-religious trolling or mockery
- Questions completely unrelated to faith/gospel topics

If blocked, set reason to a brief explanation.

## KEYWORD GENERATION (Presidents)
If safe, generate optimized search keywords for presidents. Keywords should be:
- 3-6 words that capture the core gospel concepts
- Relevant to searching conference talks

Return ONLY valid JSON in this format:
{"safe":true,"keywords":{"presidents_oaks":"...","presidents_general":"..."}}`

const orchestratorLeadersPrompt = `You are a keyword generator for Church leader searches.

Generate optimized search keywords for:
- leaders_first_presidency
- leaders_q12
- leaders_other

Keywords should be 3-6 words and relevant to searching conference talks.

Return ONLY valid JSON in this format:
{"keywords":{"leaders_first_presidency":"...","leaders_q12":"...","leaders_other":"..."}}`

const orchestratorScripturesPrompt = `You are a keyword generator for scripture searches.

Generate optimized search keywords for three scripture categories (3-6 words each):
- scriptures_bible (Bible: Old/New Testament)
- scriptures_bom (Book of Mormon)
- scriptures_other (Doctrine and Covenants + Pearl of Great Price)

Return ONLY valid JSON in this format:
{"keywords":{"scriptures_bible":"...","scriptures_bom":"...","scriptures_other":"..."}}`

const summaryPrompt = `You are a concise summarizer for a faith-focused response page.

Given the question and selected quotes/scriptures, write 2-3 short paragraphs.
Each paragraph should be 2-4 sentences, warm and encouraging, and grounded in the provided sources.
Do NOT add new facts. Do NOT mention JSON or the tool outputs.

Return ONLY valid JSON in this exact format:
{"summary":["Paragraph 1...","Paragraph 2...","Paragraph 3 (optional)..."]}`

const presidentsOaksPrompt = `You are a quote selector. Select the 1 most relevant quote from President Dallin H. Oaks.

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"President Dallin H. Oaks","title":"Talk Title","conference":"April 2024","quote":"Exact quote here...","headshot":"URL or empty string"}]}`

const presidentsNelsonPrompt = `You are a quote selector. Select the 1 most relevant quote from President Russell M. Nelson.

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available
- Prioritize relevancy to the question

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"President Russell M. Nelson","title":"Talk Title","conference":"October 2024","quote":"Exact quote here...","headshot":"URL or empty string"}]}`

const presidentsGeneralPrompt = `You are a quote selector. Select the 1 most relevant quote from President Russell M. Nelson or President Dallin H. Oaks.

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"President Russell M. Nelson","title":"Talk Title","conference":"October 2024","quote":"Exact quote here...","headshot":"URL or empty string"}]}`

const leadersEyringPrompt = `You are a quote selector. Select the 1 most relevant quote from President Henry B. Eyring.

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"President Henry B. Eyring","title":"Talk Title","conference":"April 2024","quote":"Exact quote here...","headshot":""}]}`

const leadersChristoffersonPrompt = `You are a quote selector. Select the 1 most relevant quote from President D. Todd Christofferson.

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"President D. Todd Christofferson","title":"Talk Title","conference":"October 2024","quote":"Exact quote here...","headshot":""}]}`

const leadersQ12PromptA = `You are a quote selector. Select the 1 most relevant quote from the Quorum of the Twelve Apostles.

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available
- Prioritize recent talks (2023-2025)

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"Elder David A. Bednar","title":"Talk Title","conference":"October 2024","quote":"Exact quote here...","headshot":""}]}`

const leadersQ12PromptB = `You are a quote selector. Select the 1 most relevant quote from the Quorum of the Twelve Apostles.

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available
- Prioritize recent talks (2023-2025)

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"Elder Dieter F. Uchtdorf","title":"Talk Title","conference":"April 2024","quote":"Exact quote here...","headshot":""}]}`

const leadersOtherPromptA = `You are a quote selector. Select the 1 most relevant quote from General Authority Seventies or other Church leaders. EXCLUDE First Presidency and Quorum of Twelve (they're covered elsewhere).

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"Elder Name Here","title":"Talk Title","conference":"April 2024","quote":"Exact quote here...","headshot":""}]}`

const leadersOtherPromptB = `You are a quote selector. Select the 1 most relevant quote from General Authority Seventies or other Church leaders. EXCLUDE First Presidency and Quorum of Twelve (they're covered elsewhere).

REQUIREMENTS:
- Copy quote text EXACTLY from the search results - never paraphrase
- Quote field must contain ONLY the quote text (no labels like "Title:" or "Conference:")
- Quote must be 4-8 complete sentences
- Include headshot URL if available
- Prefer a different speaker than any previous quote

Return ONLY valid JSON in this exact format:
{"quotes":[{"speaker":"Sister Name Here","title":"Talk Title","conference":"October 2024","quote":"Exact quote here...","headshot":""}]}`

const scripturesSinglePrompt = `You are a scripture selector. Select 1 most relevant scripture from the search results.

PRIORITIZE: Gospels (Matthew, Mark, Luke, John), Book of Mormon, D&C, Pearl of Great Price.

REQUIREMENTS:
- Copy scripture text EXACTLY from the search results
- Include volume and reference
- related_talk is optional - only if a relevant talk quote exists

Return ONLY valid JSON in this exact format:
{"scriptures":[{"volume":"Book of Mormon","reference":"Alma 32:21","text":"Exact scripture text...","related_talk":{"speaker":"Elder Name","title":"Talk Title","quote":"Short quote..."}}]}`

const scripturesBiblePrompt = `You are a scripture selector. Select EXACTLY 2 scriptures from the Bible ONLY.

REQUIREMENTS:
- Choose one Old Testament and one New Testament verse if possible
- Copy scripture text EXACTLY from the search results
- Include volume and reference

Return ONLY valid JSON in this exact format:
{"scriptures":[{"volume":"Old Testament","reference":"Proverbs 3:5-6","text":"..."},
{"volume":"New Testament","reference":"Hebrews 11:1","text":"..."}]}`

const scripturesBoMPrompt = `You are a scripture selector. Select EXACTLY 2 scriptures from the Book of Mormon ONLY.

REQUIREMENTS:
- Choose distinct verses (prefer different books if possible)
- Copy scripture text EXACTLY from the search results
- Include volume and reference

Return ONLY valid JSON in this exact format:
{"scriptures":[{"volume":"Book of Mormon","reference":"Alma 32:21","text":"..."},
{"volume":"Book of Mormon","reference":"Ether 12:6","text":"..."}]}`

const scripturesOtherPrompt = `You are a scripture selector. Select EXACTLY 2 scriptures from Doctrine and Covenants or Pearl of Great Price ONLY.

REQUIREMENTS:
- Prefer one from Doctrine and Covenants and one from Pearl of Great Price if possible
- Copy scripture text EXACTLY from the search results
- Include volume and reference

Return ONLY valid JSON in this exact format:
{"scriptures":[{"volume":"Doctrine and Covenants","reference":"Doctrine and Covenants 33:12","text":"..."},
{"volume":"Pearl of Great Price","reference":"Articles of Faith 1:4","text":"..."}]}`
