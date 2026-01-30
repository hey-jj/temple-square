# L8 Implementation Prompt: Gemini REST API Integration

## Role
L8 Principal Engineer - Implementation Executor

## Goal
Replace the broken ADK/genai library integration with direct REST calls to Gemini API. The genai library has a bug where it ignores BackendGeminiAPI in Cloud Run environments. We bypass this entirely with raw HTTP.

## Hard Requirements
1. Use `gemini-3-flash-preview` model via REST API
2. Endpoint: `https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:streamGenerateContent`
3. Auth: `x-goog-api-key` header with `GEMINI_API_KEY` env var
4. Structured JSON output via `responseMimeType: "application/json"` and `responseJsonSchema`
5. Keep MCP Toolbox integration for database tools
6. Maintain parallel execution of 3 agents (presidents, leaders, scriptures)
7. SSE streaming to frontend must continue working
8. Fail-fast - no silent fallbacks

## Reference: Working Gemini REST Call
```bash
curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent" \
  -H "x-goog-api-key: $GEMINI_API_KEY" \
  -H 'Content-Type: application/json' \
  -X POST \
  -d '{
    "contents": [{"parts": [{"text": "..."}]}],
    "generationConfig": {
        "responseMimeType": "application/json",
        "responseJsonSchema": {...}
    }
  }'
```

For streaming, use `:streamGenerateContent?alt=sse` endpoint.

## Architecture
```
/ask (POST)
  -> content validation
  -> return StreamContainer HTML

/api/stream (GET)
  -> Create 3 goroutines (parallel):
     - Presidents: REST call to Gemini + MCP tools
     - Leaders: REST call to Gemini + MCP tools
     - Scriptures: REST call to Gemini + MCP tools
  -> Stream SSE events as each completes
  -> Each returns structured JSON per schema
```

## Files to Modify
1. `internal/agent/agent.go` - Replace gemini.NewModel with REST client
2. `internal/agent/gemini_client.go` - NEW: Raw HTTP client for Gemini API
3. `cmd/server/sse.go` - Update to use new agent interface
4. Remove ADK imports, keep MCP Toolbox imports

## Tasks
1. Create `internal/agent/gemini_client.go`:
   - `GeminiClient` struct with `http.Client` and API key
   - `GenerateContent(ctx, prompt, schema)` method
   - `StreamGenerateContent(ctx, prompt, schema)` method returning channel
   - Proper error handling for API errors

2. Update `internal/agent/agent.go`:
   - Remove `google.golang.org/adk` imports
   - Remove `google.golang.org/genai` imports
   - Keep `github.com/googleapis/mcp-toolbox-sdk-go/tbadk`
   - Create parallel agent structure using goroutines + channels
   - Each sub-agent: load tools from MCP Toolbox, call Gemini with tools

3. Update SSE handler to work with new agent

4. Update go.mod - remove adk/genai dependencies

5. Test locally with `make dev`

6. Deploy to Cloud Run

## Validation
- `go build ./...` succeeds
- Local test returns structured JSON for all 3 sections
- Cloud Run deployment succeeds
- Live URL returns working responses
- Logs show calls to `generativelanguage.googleapis.com`

## Acceptance Criteria
- [ ] No ADK/genai library usage
- [ ] Direct REST to generativelanguage.googleapis.com
- [ ] gemini-3-flash-preview model works
- [ ] All 3 sections return data
- [ ] SSE streaming works
- [ ] Deployed and live on Cloud Run

## Report
Write your implementation report and any blockers to:
`/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-gemini-rest-implementation-01-report.md`

## Conventional Commit
`refactor(agent): replace ADK with direct Gemini REST API`

## Go/No-Go
- Go: All acceptance criteria met, live on Cloud Run
- No-Go: Report specific blocker with evidence
