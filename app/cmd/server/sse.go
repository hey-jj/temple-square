// cmd/server/sse.go
// SSE handler for parallel agent architecture with structured outputs
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	prophetagent "github.com/temple-square/prophet-agent/internal/agent"
	"github.com/temple-square/prophet-agent/internal/ui/components"
)

// StructuredQuote matches the JSON schema from the agent
type StructuredQuote struct {
	Speaker    string `json:"speaker"`
	Title      string `json:"title"`
	Conference string `json:"conference"`
	Quote      string `json:"quote"`
	Headshot   string `json:"headshot,omitempty"`
}

// StructuredScripture matches the JSON schema from the agent
type StructuredScripture struct {
	Volume      string            `json:"volume"`
	Reference   string            `json:"reference"`
	Text        string            `json:"text"`
	RelatedTalk *RelatedTalkQuote `json:"related_talk,omitempty"`
}

// RelatedTalkQuote is a smaller quote from a talk
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

// SummaryResponse is the structured output for summary agent
type SummaryResponse struct {
	Summary []string `json:"summary"`
}

const sessionTTL = 10 * time.Minute

var (
	sessionMu      sync.Mutex
	sessionExpires = map[string]time.Time{}
)

func claimSession(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	now := time.Now()
	sessionMu.Lock()
	defer sessionMu.Unlock()
	if exp, ok := sessionExpires[sessionID]; ok {
		if exp.After(now) {
			return true
		}
		delete(sessionExpires, sessionID)
	}
	sessionExpires[sessionID] = now.Add(sessionTTL)
	return false
}

func markSessionDone(sessionID string) {
	if sessionID == "" {
		return
	}
	sessionMu.Lock()
	sessionExpires[sessionID] = time.Now().Add(sessionTTL)
	sessionMu.Unlock()
}

// handleSSEStream handles SSE streaming for parallel agent architecture
func handleSSEStream(w http.ResponseWriter, r *http.Request, agent *prophetagent.ProphetAgent) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	question := r.URL.Query().Get("q")
	sessionID := r.URL.Query().Get("session")

	if question == "" {
		sendSSEError(w, flusher, "missing question")
		return
	}
	if claimSession(sessionID) {
		log.Printf("SSE: Duplicate session, closing session=%s question=%q remote=%s", sessionID, question, r.RemoteAddr)
		sendSSEDone(w, flusher)
		return
	}
	defer markSessionDone(sessionID)
	log.Printf("SSE: Open session=%s question=%q remote=%s", sessionID, question, r.RemoteAddr)

	// Classify content (defense in depth)
	classification := prophetagent.ClassifyContent(question)
	if classification != prophetagent.ContentSafe {
		redirect := prophetagent.GetRedirectResponse(classification)
		var buf bytes.Buffer
		if err := components.RedirectResponse(redirect.Message, redirect.SuggestedQuestions).Render(ctx, &buf); err != nil {
			sendSSEError(w, flusher, "Error rendering response")
			return
		}
		fmt.Fprintf(w, "event: server-error\ndata: %s\n\n", escapeSSEData(buf.String()))
		flusher.Flush()
		sendSSEDone(w, flusher)
		return
	}

	// Run parallel agents
	log.Printf("SSE: Starting parallel agent execution for question: %s", question)
	results := agent.Run(ctx, question)

	var presidentsQuotes []StructuredQuote
	var leadersQuotes []StructuredQuote
	var bibleScriptures []StructuredScripture
	var bomScriptures []StructuredScripture
	var otherScriptures []StructuredScripture

	// Process results as they come in
	for result := range results {
		if result.Error != nil {
			log.Printf("SSE: Agent %s error: %v", result.AgentName, result.Error)
			sendSSEError(w, flusher, fmt.Sprintf("Agent %s failed: %v", result.AgentName, result.Error))
			continue
		}

		if result.Content == "" {
			log.Printf("SSE: Agent %s returned empty content", result.AgentName)
			continue
		}

		switch result.AgentName {
		case "presidents_agent":
			quotes, err := parseQuotesFromContent(result.Content)
			if err != nil {
				log.Printf("SSE: Failed to parse presidents result: %v", err)
				sendSSEError(w, flusher, "Presidents section returned malformed JSON")
				continue
			}
			presidentsQuotes = mergeUniqueQuotes(presidentsQuotes, quotes)
			if len(presidentsQuotes) > 0 {
				if err := sendPresidentsSection(ctx, w, flusher, presidentsQuotes); err != nil {
					log.Printf("SSE: Failed to render presidents section: %v", err)
				}
			}

		case "leaders_agent":
			quotes, err := parseLeadersFromContent(result.Content)
			if err != nil {
				log.Printf("SSE: Failed to parse leaders result: %v", err)
				sendSSEError(w, flusher, "Leaders section returned malformed JSON")
				continue
			}
			leadersQuotes = mergeUniqueQuotes(leadersQuotes, quotes)
			if len(leadersQuotes) > 0 {
				if err := sendLeadersSection(ctx, w, flusher, leadersQuotes); err != nil {
					log.Printf("SSE: Failed to render leaders section: %v", err)
				}
			}

		case "scriptures_bible":
			items, err := parseScripturesFromContent(result.Content)
			if err != nil {
				log.Printf("SSE: Failed to parse bible scriptures result: %v", err)
				sendSSEError(w, flusher, "Scripture section returned malformed JSON")
				continue
			}
			bibleScriptures = mergeUniqueScriptures(bibleScriptures, items)
			if err := sendScripturesSection(ctx, w, flusher, bibleScriptures, bomScriptures, otherScriptures); err != nil {
				log.Printf("SSE: Failed to render scriptures section: %v", err)
			}

		case "scriptures_bom":
			items, err := parseScripturesFromContent(result.Content)
			if err != nil {
				log.Printf("SSE: Failed to parse Book of Mormon result: %v", err)
				sendSSEError(w, flusher, "Scripture section returned malformed JSON")
				continue
			}
			bomScriptures = mergeUniqueScriptures(bomScriptures, items)
			if err := sendScripturesSection(ctx, w, flusher, bibleScriptures, bomScriptures, otherScriptures); err != nil {
				log.Printf("SSE: Failed to render scriptures section: %v", err)
			}

		case "scriptures_other":
			items, err := parseScripturesFromContent(result.Content)
			if err != nil {
				log.Printf("SSE: Failed to parse other scriptures result: %v", err)
				sendSSEError(w, flusher, "Scripture section returned malformed JSON")
				continue
			}
			otherScriptures = mergeUniqueScriptures(otherScriptures, items)
			if err := sendScripturesSection(ctx, w, flusher, bibleScriptures, bomScriptures, otherScriptures); err != nil {
				log.Printf("SSE: Failed to render scriptures section: %v", err)
			}

		default:
			log.Printf("DEBUG: Ignoring content from unknown agent: %s", result.AgentName)
		}
	}

	// Final summary (2-3 paragraphs)
	allScriptures := append(append([]StructuredScripture{}, bibleScriptures...), bomScriptures...)
	allScriptures = append(allScriptures, otherScriptures...)
	summaryContent, err := agent.GenerateSummary(ctx, question,
		toAgentQuotes(presidentsQuotes),
		toAgentQuotes(leadersQuotes),
		toAgentScriptures(allScriptures))
	if err != nil {
		log.Printf("SSE: Summary generation failed: %v", err)
	} else if summaryContent != "" {
		paras, err := parseSummaryFromContent(summaryContent)
		if err != nil {
			log.Printf("SSE: Failed to parse summary: %v", err)
		} else if len(paras) > 0 {
			if err := sendSummarySection(ctx, w, flusher, paras); err != nil {
				log.Printf("SSE: Failed to render summary section: %v", err)
			}
		}
	}

	sendSSEDone(w, flusher)
	log.Printf("SSE: Completed streaming for question: %s", question)
}

func parseQuotesFromContent(content string) ([]StructuredQuote, error) {
	jsonContent, err := extractFirstJSON(content)
	if err != nil {
		return nil, err
	}

	var resp PresidentsResponse
	if err := json.Unmarshal([]byte(jsonContent), &resp); err != nil {
		return nil, err
	}
	return resp.Quotes, nil
}

func parseLeadersFromContent(content string) ([]StructuredQuote, error) {
	jsonContent, err := extractFirstJSON(content)
	if err != nil {
		return nil, err
	}

	var resp LeadersResponse
	if err := json.Unmarshal([]byte(jsonContent), &resp); err != nil {
		return nil, err
	}
	return resp.Quotes, nil
}

func parseScripturesFromContent(content string) ([]StructuredScripture, error) {
	jsonContent, err := extractFirstJSON(content)
	if err != nil {
		return nil, err
	}

	var resp ScripturesResponse
	if err := json.Unmarshal([]byte(jsonContent), &resp); err != nil {
		return nil, err
	}
	return resp.Scriptures, nil
}

func parseSummaryFromContent(content string) ([]string, error) {
	jsonContent, err := extractFirstJSON(content)
	if err != nil {
		return nil, err
	}

	var resp SummaryResponse
	if err := json.Unmarshal([]byte(jsonContent), &resp); err != nil {
		return nil, err
	}
	return resp.Summary, nil
}

func mergeUniqueQuotes(existing, incoming []StructuredQuote) []StructuredQuote {
	seen := make(map[string]struct{}, len(existing))
	for _, q := range existing {
		seen[quoteKey(q)] = struct{}{}
	}
	for _, q := range incoming {
		key := quoteKey(q)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		existing = append(existing, q)
	}
	return existing
}

func mergeUniqueScriptures(existing, incoming []StructuredScripture) []StructuredScripture {
	seen := make(map[string]struct{}, len(existing))
	for _, s := range existing {
		seen[scriptureKey(s)] = struct{}{}
	}
	for _, s := range incoming {
		key := scriptureKey(s)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		existing = append(existing, s)
	}
	return existing
}

func quoteKey(q StructuredQuote) string {
	return fmt.Sprintf("%s|%s|%s|%s", q.Speaker, q.Title, q.Conference, q.Quote)
}

func scriptureKey(s StructuredScripture) string {
	return fmt.Sprintf("%s|%s|%s", s.Volume, s.Reference, s.Text)
}

func sendPresidentsSection(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, quotes []StructuredQuote) error {
	orderedQuotes := append([]StructuredQuote(nil), quotes...)
	sortPresidentsQuotes(orderedQuotes)
	speakers := convertQuotesToSpeakers(orderedQuotes)
	var buf bytes.Buffer
	if err := components.PresidentsSection(speakers).Render(ctx, &buf); err != nil {
		return err
	}
	fmt.Fprintf(w, "event: presidents\ndata: %s\n\n", escapeSSEData(buf.String()))
	flusher.Flush()
	return nil
}

func sortPresidentsQuotes(quotes []StructuredQuote) {
	sort.SliceStable(quotes, func(i, j int) bool {
		return oaksPriority(quotes[i]) < oaksPriority(quotes[j])
	})
}

func oaksPriority(q StructuredQuote) int {
	if isOaksSpeaker(q.Speaker) {
		return 0
	}
	return 1
}

func isOaksSpeaker(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	return strings.Contains(name, "oaks")
}

func sendLeadersSection(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, quotes []StructuredQuote) error {
	speakers := convertQuotesToSpeakers(quotes)
	var buf bytes.Buffer
	if err := components.LeadersSection(speakers).Render(ctx, &buf); err != nil {
		return err
	}
	fmt.Fprintf(w, "event: leaders\ndata: %s\n\n", escapeSSEData(buf.String()))
	flusher.Flush()
	return nil
}

func sendScripturesSection(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, bible []StructuredScripture, bom []StructuredScripture, other []StructuredScripture) error {
	bibleCards := convertStructuredScriptures(bible)
	bomCards := convertStructuredScriptures(bom)
	otherCards := convertStructuredScriptures(other)
	var buf bytes.Buffer
	if err := components.ScripturesSection(bibleCards, bomCards, otherCards).Render(ctx, &buf); err != nil {
		return err
	}
	fmt.Fprintf(w, "event: scriptures\ndata: %s\n\n", escapeSSEData(buf.String()))
	flusher.Flush()
	return nil
}

func sendSummarySection(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, paragraphs []string) error {
	var buf bytes.Buffer
	if err := components.SummarySection(paragraphs).Render(ctx, &buf); err != nil {
		return err
	}
	fmt.Fprintf(w, "event: summary\ndata: %s\n\n", escapeSSEData(buf.String()))
	flusher.Flush()
	return nil
}

// extractFirstJSON finds and extracts the first complete JSON object from a string.
// This is needed because the agent may produce multiple JSON objects concatenated together
// (e.g., from multiple tool calls or turns).
func extractFirstJSON(content string) (string, error) {
	// Find the first '{' - start of JSON object
	start := strings.Index(content, "{")
	if start == -1 {
		return "", fmt.Errorf("no JSON object found")
	}

	// Track brace depth to find matching '}'
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(content); i++ {
		c := content[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return content[start : i+1], nil
			}
		}
	}

	return "", fmt.Errorf("incomplete JSON object")
}

// tryParseAndSendSection attempts to parse JSON and send the rendered section.
// Returns (sent bool, err error) - fail-fast on malformed JSON.
func tryParseAndSendSection(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, agentName, content string) (bool, error) {
	var buf bytes.Buffer
	var eventName string

	// Extract the first complete JSON object from the content
	// This handles cases where multiple JSON objects are concatenated
	jsonContent, err := extractFirstJSON(content)
	if err != nil {
		// Log but don't error - might still be accumulating
		log.Printf("DEBUG: Cannot extract JSON from %s: %v", agentName, err)
		return false, nil
	}
	content = jsonContent

	// Truncate content for logging (max 200 chars)
	contentSnippet := content
	if len(contentSnippet) > 200 {
		contentSnippet = contentSnippet[:200] + "..."
	}

	switch agentName {
	case "presidents_agent":
		eventName = "presidents"
		var resp PresidentsResponse
		if err := json.Unmarshal([]byte(content), &resp); err != nil {
			errMsg := fmt.Sprintf("JSON parse error from %s: %v", agentName, err)
			log.Printf("ERROR: %s | Content: %s", errMsg, contentSnippet)
			sendSSEError(w, flusher, fmt.Sprintf("Agent %s returned malformed JSON - this is an agent configuration error", agentName))
			return false, fmt.Errorf("%s", errMsg)
		}
		if len(resp.Quotes) == 0 {
			log.Printf("WARN: %s returned empty quotes array | Content: %s", agentName, contentSnippet)
			return false, nil // Empty is valid, just nothing to render
		}
		speakers := convertQuotesToSpeakers(resp.Quotes)
		if err := components.PresidentsSection(speakers).Render(ctx, &buf); err != nil {
			errMsg := fmt.Sprintf("Render error for %s: %v", agentName, err)
			log.Printf("ERROR: %s", errMsg)
			sendSSEError(w, flusher, fmt.Sprintf("Failed to render %s section", eventName))
			return false, fmt.Errorf("%s", errMsg)
		}

	case "leaders_agent":
		eventName = "leaders"
		var resp LeadersResponse
		if err := json.Unmarshal([]byte(content), &resp); err != nil {
			errMsg := fmt.Sprintf("JSON parse error from %s: %v", agentName, err)
			log.Printf("ERROR: %s | Content: %s", errMsg, contentSnippet)
			sendSSEError(w, flusher, fmt.Sprintf("Agent %s returned malformed JSON - this is an agent configuration error", agentName))
			return false, fmt.Errorf("%s", errMsg)
		}
		if len(resp.Quotes) == 0 {
			log.Printf("WARN: %s returned empty quotes array | Content: %s", agentName, contentSnippet)
			return false, nil // Empty is valid, just nothing to render
		}
		speakers := convertQuotesToSpeakers(resp.Quotes)
		if err := components.LeadersSection(speakers).Render(ctx, &buf); err != nil {
			errMsg := fmt.Sprintf("Render error for %s: %v", agentName, err)
			log.Printf("ERROR: %s", errMsg)
			sendSSEError(w, flusher, fmt.Sprintf("Failed to render %s section", eventName))
			return false, fmt.Errorf("%s", errMsg)
		}

	case "scriptures_agent":
		eventName = "scriptures"
		var resp ScripturesResponse
		if err := json.Unmarshal([]byte(content), &resp); err != nil {
			errMsg := fmt.Sprintf("JSON parse error from %s: %v", agentName, err)
			log.Printf("ERROR: %s | Content: %s", errMsg, contentSnippet)
			sendSSEError(w, flusher, fmt.Sprintf("Agent %s returned malformed JSON - this is an agent configuration error", agentName))
			return false, fmt.Errorf("%s", errMsg)
		}
		if len(resp.Scriptures) == 0 {
			log.Printf("WARN: %s returned empty scriptures array | Content: %s", agentName, contentSnippet)
			return false, nil // Empty is valid, just nothing to render
		}
		scriptures := convertStructuredScriptures(resp.Scriptures)
		if err := components.ScripturesSection(scriptures, nil, nil).Render(ctx, &buf); err != nil {
			errMsg := fmt.Sprintf("Render error for %s: %v", agentName, err)
			log.Printf("ERROR: %s", errMsg)
			sendSSEError(w, flusher, fmt.Sprintf("Failed to render %s section", eventName))
			return false, fmt.Errorf("%s", errMsg)
		}

	default:
		// Unknown agent - log but don't error (might be orchestrator or other internal agent)
		log.Printf("DEBUG: Ignoring content from unknown agent: %s", agentName)
		return false, nil
	}

	if buf.Len() > 0 {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, escapeSSEData(buf.String()))
		flusher.Flush()
		log.Printf("Streamed %s section (%d bytes)", eventName, buf.Len())
		return true, nil
	}

	return false, nil
}

// convertQuotesToSpeakers converts structured quotes to component speakers
func convertQuotesToSpeakers(quotes []StructuredQuote) []components.SpeakerQuote {
	speakers := make([]components.SpeakerQuote, len(quotes))
	allowedPrefix := assetsBaseURL + "/headshots/"
	for i, q := range quotes {
		q = sanitizeStructuredQuote(q)
		headshot := lookupSpeakerHeadshot(q.Speaker)
		if headshot == "" && isValidHeadshotURL(q.Headshot, allowedPrefix) {
			headshot = q.Headshot
		}
		speakers[i] = components.SpeakerQuote{
			Name:       q.Speaker,
			TalkTitle:  q.Title,
			Conference: q.Conference,
			Quotes:     []string{q.Quote},
			Headshot:   headshot,
		}
	}
	return speakers
}

func isValidHeadshotURL(url string, allowedPrefix string) bool {
	if url == "" {
		return false
	}
	if !strings.HasPrefix(url, allowedPrefix) {
		return false
	}
	if !strings.HasSuffix(url, ".webp") {
		return false
	}
	if strings.ContainsAny(url, " \t\r\n\"'<>") {
		return false
	}
	return true
}

func sanitizeStructuredQuote(q StructuredQuote) StructuredQuote {
	raw := strings.Join([]string{q.Speaker, q.Title, q.Conference, q.Quote, q.Headshot}, " ")
	if val, ok := extractEmbeddedJSONValue(raw, "speaker"); ok {
		q.Speaker = val
	}
	if val, ok := extractEmbeddedJSONValue(raw, "title"); ok {
		q.Title = val
	}
	if val, ok := extractEmbeddedJSONValue(raw, "conference"); ok {
		q.Conference = val
	}
	if val, ok := extractEmbeddedJSONValue(raw, "quote"); ok {
		q.Quote = val
	}
	if val, ok := extractEmbeddedJSONValue(raw, "headshot"); ok {
		q.Headshot = val
	}
	q.Title = trimAtJSONKey(q.Title, "conference", "quote", "headshot")
	q.Conference = trimAtJSONKey(q.Conference, "quote", "headshot", "title")
	q.Headshot = trimAtJSONKey(q.Headshot, "conference", "quote", "title")
	q.Speaker = sanitizeLabelPrefix(q.Speaker, "Speaker:")
	q.Title = sanitizeBetweenLabels(q.Title, "Title:", "Conference:", "Quote:", "Headshot:")
	q.Conference = sanitizeBetweenLabels(q.Conference, "Conference:", "Quote:", "Headshot:")
	q.Quote = sanitizeQuoteText(q.Quote)
	q.Headshot = sanitizeLabelPrefix(q.Headshot, "Headshot:")
	return q
}

func sanitizeQuoteText(text string) string {
	cleaned := strings.TrimSpace(text)
	if extracted, ok := extractEmbeddedJSONValue(cleaned, "quote"); ok {
		return extracted
	}
	if idx := strings.LastIndex(cleaned, "Quote:"); idx != -1 {
		cleaned = cleaned[idx+len("Quote:"):]
	}
	if idx := strings.Index(cleaned, "Headshot:"); idx != -1 {
		cleaned = cleaned[:idx]
	}
	return strings.TrimSpace(cleaned)
}

func sanitizeLabelPrefix(text, label string) string {
	cleaned := strings.TrimSpace(text)
	if strings.HasPrefix(cleaned, label) {
		return strings.TrimSpace(strings.TrimPrefix(cleaned, label))
	}
	return cleaned
}

func sanitizeBetweenLabels(text string, primary string, stops ...string) string {
	cleaned := strings.TrimSpace(text)
	if idx := strings.LastIndex(cleaned, primary); idx != -1 {
		cleaned = cleaned[idx+len(primary):]
	}
	for _, stop := range stops {
		if idx := strings.Index(cleaned, stop); idx != -1 {
			cleaned = cleaned[:idx]
		}
	}
	return strings.TrimSpace(cleaned)
}

func extractEmbeddedJSONValue(text, key string) (string, bool) {
	patterns := []struct {
		start   string
		escaped bool
	}{
		{start: fmt.Sprintf("\"%s\":\"", key), escaped: false},
		{start: fmt.Sprintf("\"%s\": \"", key), escaped: false},
		{start: fmt.Sprintf("\\\\\"%s\\\\\":\\\\\"", key), escaped: true},
		{start: fmt.Sprintf("\\\\\"%s\\\\\": \\\\\"", key), escaped: true},
	}

	for _, pattern := range patterns {
		idx := strings.LastIndex(text, pattern.start)
		if idx == -1 {
			continue
		}
		rest := text[idx+len(pattern.start):]
		if pattern.escaped {
			end := strings.Index(rest, "\\\\\"")
			if end == -1 {
				continue
			}
			val := rest[:end]
			val = strings.ReplaceAll(val, "\\\\n", "\n")
			val = strings.ReplaceAll(val, "\\\\\"", "\"")
			return strings.TrimSpace(val), true
		}

		for i := 0; i < len(rest); i++ {
			if rest[i] == '"' {
				if i > 0 && rest[i-1] == '\\' {
					continue
				}
				return strings.TrimSpace(rest[:i]), true
			}
		}
	}

	return "", false
}

func trimAtJSONKey(text string, keys ...string) string {
	cleaned := strings.TrimSpace(text)
	for _, key := range keys {
		patterns := []string{
			fmt.Sprintf("\", \"%s\"", key),
			fmt.Sprintf("\", \"%s\":", key),
			fmt.Sprintf("\"%s\":", key),
			fmt.Sprintf("\", \\\"%s\\\"", key),
			fmt.Sprintf("\", \\\"%s\\\":", key),
		}
		for _, pattern := range patterns {
			if idx := strings.Index(cleaned, pattern); idx != -1 {
				cleaned = cleaned[:idx]
				break
			}
		}
	}
	cleaned = strings.TrimSuffix(cleaned, "\"")
	return strings.TrimSpace(cleaned)
}

// convertStructuredScriptures converts to component scriptures with related talks
func convertStructuredScriptures(scriptures []StructuredScripture) []components.ScriptureWithTalk {
	result := make([]components.ScriptureWithTalk, len(scriptures))
	for i, s := range scriptures {
		result[i] = components.ScriptureWithTalk{
			ScriptureRef: components.ScriptureRef{
				Volume:    s.Volume,
				Reference: s.Reference,
				Text:      s.Text,
			},
		}
		if s.RelatedTalk != nil {
			result[i].RelatedTalk = &components.TalkPullQuote{
				Speaker: s.RelatedTalk.Speaker,
				Title:   s.RelatedTalk.Title,
				Quote:   s.RelatedTalk.Quote,
			}
		}
	}
	return result
}

func toAgentQuotes(quotes []StructuredQuote) []prophetagent.StructuredQuote {
	out := make([]prophetagent.StructuredQuote, len(quotes))
	for i, q := range quotes {
		out[i] = prophetagent.StructuredQuote{
			Speaker:    q.Speaker,
			Title:      q.Title,
			Conference: q.Conference,
			Quote:      q.Quote,
			Headshot:   q.Headshot,
		}
	}
	return out
}

func toAgentScriptures(items []StructuredScripture) []prophetagent.StructuredScripture {
	out := make([]prophetagent.StructuredScripture, len(items))
	for i, s := range items {
		var related *prophetagent.RelatedTalkQuote
		if s.RelatedTalk != nil {
			related = &prophetagent.RelatedTalkQuote{
				Speaker: s.RelatedTalk.Speaker,
				Title:   s.RelatedTalk.Title,
				Quote:   s.RelatedTalk.Quote,
			}
		}
		out[i] = prophetagent.StructuredScripture{
			Volume:      s.Volume,
			Reference:   s.Reference,
			Text:        s.Text,
			RelatedTalk: related,
		}
	}
	return out
}

// sendSSEError sends an error event
func sendSSEError(w http.ResponseWriter, flusher http.Flusher, message string) {
	fmt.Fprintf(w, "event: server-error\ndata: <div class=\"text-red-600\">Error: %s</div>\n\n", escapeSSEData(message))
	flusher.Flush()
}

// sendSSEDone sends the done event
func sendSSEDone(w http.ResponseWriter, flusher http.Flusher) {
	// Note: htmx-ext-sse requires non-empty data to avoid swap errors
	fmt.Fprintf(w, "event: done\ndata: complete\n\n")
	flusher.Flush()
}

// handleTestGemini tests Gemini API latency from Cloud Run
func handleTestGemini(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		http.Error(w, `{"error": "GEMINI_API_KEY not set"}`, 500)
		return
	}

	// Simple test request
	reqBody := `{
		"contents": [{"parts": [{"text": "Say hello in 5 words"}], "role": "user"}],
		"generationConfig": {"maxOutputTokens": 100}
	}`

	start := time.Now()

	client := &http.Client{Timeout: 60 * time.Second}
	req, _ := http.NewRequest("POST",
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent",
		strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(w, `{"error": "%s", "elapsed_ms": %d}`, err.Error(), elapsed.Milliseconds())
		return
	}
	defer resp.Body.Close()

	fmt.Fprintf(w, `{"status": %d, "elapsed_ms": %d}`, resp.StatusCode, elapsed.Milliseconds())
}
