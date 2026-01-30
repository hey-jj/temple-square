// Package agent provides a direct REST client for Google's Gemini API.
// This bypasses the genai library which has a bug where it ignores BackendGeminiAPI
// in Cloud Run environments.
package agent

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// GeminiAPIEndpoint is the base URL for the Gemini API
	GeminiAPIEndpoint = "https://generativelanguage.googleapis.com/v1beta"
	// DefaultModel is the model to use for generation
	// Spec: https://aistackregistry.com/latest/models/gemini/gemini-3-flash-preview/spec.json
	DefaultModel = "gemini-3-flash-preview"
)

// GeminiClient is a direct REST client for the Gemini API
type GeminiClient struct {
	httpClient *http.Client
	apiKey     string
	model      string
	trace      bool
	reqSeq     uint64
	dumpPath   string
	dumped     uint32
	dumpLog    bool
	dumpAll    bool
}

// NewGeminiClient creates a new Gemini REST client
func NewGeminiClient(apiKey string) (*GeminiClient, error) {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("GEMINI_API_KEY is required")
		}
	}
	trace := envBool("GEMINI_TRACE")
	dumpPath := strings.TrimSpace(os.Getenv("GEMINI_DUMP_PATH"))
	dumpLog := envBool("GEMINI_DUMP_LOG")
	dumpAll := envBool("GEMINI_DUMP_ALL")
	transport := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		ForceAttemptHTTP2:   true,
	}
	return &GeminiClient{
		httpClient: &http.Client{
			Timeout:   240 * time.Second, // 4 min to allow long format passes
			Transport: transport,
		},
		apiKey:   apiKey,
		model:    DefaultModel,
		trace:    trace,
		dumpPath: dumpPath,
		dumpLog:  dumpLog,
		dumpAll:  dumpAll,
	}, nil
}

// Content represents a message in the conversation
type Content struct {
	Parts []*Part `json:"parts"`
	Role  string  `json:"role"`
}

// Part represents a piece of content
type Part struct {
	Text             string        `json:"text,omitempty"`
	ThoughtSignature string        `json:"thoughtSignature,omitempty"` // Required for thinking models
	FunctionCall     *FunctionCall `json:"functionCall,omitempty"`
	FunctionResp     *FunctionResp `json:"functionResponse,omitempty"`
}

// FunctionCall represents a function call from the model
type FunctionCall struct {
	Name             string         `json:"name"`
	Args             map[string]any `json:"args"`
	ThoughtSignature string         `json:"-"` // Populated from parent Part, not JSON
}

// FunctionResp represents a function response to the model
type FunctionResp struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

// Tool represents a tool available to the model
type Tool struct {
	FunctionDeclarations []*FunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// FunctionDeclaration describes a function the model can call
type FunctionDeclaration struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters,omitempty"`
}

// GenerationConfig contains configuration for content generation
type GenerationConfig struct {
	Temperature        *float32        `json:"temperature,omitempty"`
	MaxOutputTokens    int             `json:"maxOutputTokens,omitempty"`
	ResponseMIMEType   string          `json:"responseMimeType,omitempty"`
	ResponseJSONSchema any             `json:"responseJsonSchema,omitempty"`
	ThinkingConfig     *ThinkingConfig `json:"thinkingConfig,omitempty"`
}

// ThinkingConfig controls Gemini thinking behavior.
type ThinkingConfig struct {
	ThinkingLevel string `json:"thinkingLevel,omitempty"`
}

// SafetySetting configures safety thresholds
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GenerateRequest is the request body for the Gemini API
type GenerateRequest struct {
	Contents         []*Content        `json:"contents"`
	SystemInstruct   *Content          `json:"systemInstruction,omitempty"`
	Tools            []*Tool           `json:"tools,omitempty"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []*SafetySetting  `json:"safetySettings,omitempty"`
}

// GenerateResponse is the response from the Gemini API
type GenerateResponse struct {
	Candidates    []*Candidate   `json:"candidates"`
	UsageMetadata *UsageMetadata `json:"usageMetadata,omitempty"`
}

// Candidate represents a response candidate
type Candidate struct {
	Content       *Content        `json:"content"`
	FinishReason  string          `json:"finishReason,omitempty"`
	SafetyRatings []*SafetyRating `json:"safetyRatings,omitempty"`
}

// SafetyRating represents a safety rating for content
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// UsageMetadata contains token usage information
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// SSEEvent represents a Server-Sent Event from streaming
type SSEEvent struct {
	Data string
}

type requestTrace struct {
	id           uint64
	host         string
	start        time.Time
	dnsStart     time.Time
	dnsDone      time.Time
	connectStart time.Time
	connectDone  time.Time
	tlsStart     time.Time
	tlsDone      time.Time
	gotConn      time.Time
	wroteRequest time.Time
	firstByte    time.Time
	reused       bool
	wasIdle      bool
	idleTime     time.Duration
	network      string
	addr         string
	dnsErr       error
	connErr      error
	tlsErr       error
	wroteReqErr  error
}

// GenerateContent makes a non-streaming request to the Gemini API
func (c *GeminiClient) GenerateContent(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent", GeminiAPIEndpoint, c.model)
	return c.doRequest(ctx, url, req)
}

// StreamGenerateContent makes a streaming request to the Gemini API
// Returns a channel that receives response chunks
func (c *GeminiClient) StreamGenerateContent(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
	respChan := make(chan *GenerateResponse, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(respChan)
		defer close(errChan)

		url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", GeminiAPIEndpoint, c.model)

		body, err := json.Marshal(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
		if err != nil {
			errChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("x-goog-api-key", c.apiKey)

		traceReq, traceData := c.attachTrace(httpReq)
		start := time.Now()
		if traceData != nil {
			traceData.start = start
		}

		resp, err := c.httpClient.Do(traceReq)
		headerElapsed := time.Since(start)
		if traceData != nil {
			traceData.log(resp, err, headerElapsed, headerElapsed)
		}
		if err != nil {
			errChan <- fmt.Errorf("request failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errChan <- fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
			return
		}

		// Parse SSE stream
		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return
				}
				errChan <- fmt.Errorf("failed to read stream: %w", err)
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// SSE format: data: {...}
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				var genResp GenerateResponse
				if err := json.Unmarshal([]byte(data), &genResp); err != nil {
					// Skip malformed JSON, might be keepalive
					continue
				}

				select {
				case <-ctx.Done():
					return
				case respChan <- &genResp:
				}
			}
		}
	}()

	return respChan, errChan
}

func (c *GeminiClient) nextReqID() uint64 {
	return atomic.AddUint64(&c.reqSeq, 1)
}

func envBool(key string) bool {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return false
	}
	switch strings.ToLower(val) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func (c *GeminiClient) attachTrace(req *http.Request) (*http.Request, *requestTrace) {
	if !c.trace {
		return req, nil
	}

	traceData := &requestTrace{
		id:   c.nextReqID(),
		host: req.URL.Host,
	}

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			traceData.dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			traceData.dnsDone = time.Now()
			traceData.dnsErr = info.Err
		},
		ConnectStart: func(network, addr string) {
			traceData.connectStart = time.Now()
			traceData.network = network
			traceData.addr = addr
		},
		ConnectDone: func(network, addr string, err error) {
			traceData.connectDone = time.Now()
			traceData.connErr = err
		},
		TLSHandshakeStart: func() {
			traceData.tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			traceData.tlsDone = time.Now()
			traceData.tlsErr = err
		},
		GotConn: func(info httptrace.GotConnInfo) {
			traceData.gotConn = time.Now()
			traceData.reused = info.Reused
			traceData.wasIdle = info.WasIdle
			traceData.idleTime = info.IdleTime
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			traceData.wroteRequest = time.Now()
			traceData.wroteReqErr = info.Err
		},
		GotFirstResponseByte: func() {
			traceData.firstByte = time.Now()
		},
	}

	return req.WithContext(httptrace.WithClientTrace(req.Context(), trace)), traceData
}

func (t *requestTrace) log(resp *http.Response, err error, headerElapsed time.Duration, totalElapsed time.Duration) {
	if t == nil {
		return
	}

	status := 0
	proto := ""
	if resp != nil {
		status = resp.StatusCode
		proto = resp.Proto
	}

	log.Printf(
		"[gemini][trace %d] host=%s status=%d proto=%s reused=%t was_idle=%t idle=%s dns=%s conn=%s tls=%s wrote=%s ttfb=%s headers=%s total=%s net=%s addr=%s dns_err=%v conn_err=%v tls_err=%v write_err=%v",
		t.id,
		t.host,
		status,
		proto,
		t.reused,
		t.wasIdle,
		t.idleTime,
		dur(t.dnsStart, t.dnsDone),
		dur(t.connectStart, t.connectDone),
		dur(t.tlsStart, t.tlsDone),
		dur(t.start, t.wroteRequest),
		dur(t.start, t.firstByte),
		headerElapsed,
		totalElapsed,
		t.network,
		t.addr,
		t.dnsErr,
		t.connErr,
		t.tlsErr,
		t.wroteReqErr,
	)
	if err != nil {
		log.Printf("[gemini][trace %d] request error: %v", t.id, err)
	}
}

func dur(start, end time.Time) time.Duration {
	if start.IsZero() || end.IsZero() {
		return 0
	}
	if end.Before(start) {
		return 0
	}
	return end.Sub(start)
}

// doRequest performs a non-streaming API request
func (c *GeminiClient) doRequest(ctx context.Context, url string, req *GenerateRequest) (*GenerateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("[gemini] Request body size: %d bytes", len(body))
	if c.dumpAll && (c.dumpPath != "" || c.dumpLog) {
		if c.dumpPath != "" {
			if err := os.WriteFile(c.dumpPath, body, 0o644); err != nil {
				log.Printf("[gemini] Failed to dump request body to %s: %v", c.dumpPath, err)
			} else {
				log.Printf("[gemini] Dumped request body to %s", c.dumpPath)
			}
		}
		if c.dumpLog {
			encoded := base64.StdEncoding.EncodeToString(body)
			log.Printf("[gemini] Dumped request body (base64): %s", encoded)
		}
	} else if (c.dumpPath != "" || c.dumpLog) && atomic.CompareAndSwapUint32(&c.dumped, 0, 1) {
		if c.dumpPath != "" {
			if err := os.WriteFile(c.dumpPath, body, 0o644); err != nil {
				log.Printf("[gemini] Failed to dump request body to %s: %v", c.dumpPath, err)
			} else {
				log.Printf("[gemini] Dumped request body to %s", c.dumpPath)
			}
		}
		if c.dumpLog {
			encoded := base64.StdEncoding.EncodeToString(body)
			log.Printf("[gemini] Dumped request body (base64): %s", encoded)
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", c.apiKey)

	traceReq, traceData := c.attachTrace(httpReq)
	start := time.Now()
	if traceData != nil {
		traceData.start = start
	}

	resp, err := c.httpClient.Do(traceReq)
	headerElapsed := time.Since(start)
	if err != nil {
		log.Printf("[gemini] Request failed after %v: %v", headerElapsed, err)
		if traceData != nil {
			traceData.log(resp, err, headerElapsed, headerElapsed)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if traceData != nil {
			traceData.log(resp, err, headerElapsed, time.Since(start))
		}
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	totalElapsed := time.Since(start)
	log.Printf("[gemini] Response headers in %v, total %v, status: %d", headerElapsed, totalElapsed, resp.StatusCode)
	if traceData != nil {
		traceData.log(resp, nil, headerElapsed, totalElapsed)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(respBody, &genResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if genResp.UsageMetadata != nil {
		log.Printf("[gemini] Usage tokens: prompt=%d candidates=%d total=%d", genResp.UsageMetadata.PromptTokenCount, genResp.UsageMetadata.CandidatesTokenCount, genResp.UsageMetadata.TotalTokenCount)
	}
	if len(genResp.Candidates) > 0 {
		log.Printf("[gemini] Finish reason: %s", genResp.Candidates[0].FinishReason)
	}

	return &genResp, nil
}

// ExtractText extracts all text from a response
func (r *GenerateResponse) ExtractText() string {
	var texts []string
	for _, candidate := range r.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					texts = append(texts, part.Text)
				}
			}
		}
	}
	return strings.Join(texts, "")
}

// ExtractFunctionCalls extracts function calls from a response
// It also captures the thoughtSignature from the Part and attaches it to the FunctionCall
// This is required for thinking models like gemini-3-flash-preview
func (r *GenerateResponse) ExtractFunctionCalls() []*FunctionCall {
	var calls []*FunctionCall
	for _, candidate := range r.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part.FunctionCall != nil {
					// Copy the thought signature from the Part to the FunctionCall
					part.FunctionCall.ThoughtSignature = part.ThoughtSignature
					calls = append(calls, part.FunctionCall)
				}
			}
		}
	}
	return calls
}

// HasFunctionCalls returns true if the response contains function calls
func (r *GenerateResponse) HasFunctionCalls() bool {
	return len(r.ExtractFunctionCalls()) > 0
}

// GetFinishReason returns the finish reason from the first candidate
func (r *GenerateResponse) GetFinishReason() string {
	if len(r.Candidates) > 0 {
		return r.Candidates[0].FinishReason
	}
	return ""
}

// DefaultSafetySettings returns permissive settings for religious content
// Using BLOCK_ONLY_HIGH to allow discussions of afterlife, death, resurrection, etc.
func DefaultSafetySettings() []*SafetySetting {
	return []*SafetySetting{
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_CIVIC_INTEGRITY", Threshold: "BLOCK_ONLY_HIGH"},
	}
}
