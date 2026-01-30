# L8 Implementation Prompt: Add thought_signature Support

## Role
L8 Principal Engineer - Implementation Executor

## Goal
Fix: "Function call is missing a thought_signature in functionCall parts"

gemini-3-flash-preview uses "thinking" which requires preserving thought_signature from function calls to function responses.

## Solution
1. Add ThoughtSignature field to Part struct
2. When receiving function calls, capture the thought_signature
3. Include thought_signature in the function response part

## Files to Modify

### 1. `/Users/justinjones/Developer/temple-square/app/internal/agent/gemini_client.go`

```go
// Part represents a piece of content
type Part struct {
    Text            string        `json:"text,omitempty"`
    ThoughtSignature string       `json:"thoughtSignature,omitempty"`  // ADD THIS
    FunctionCall    *FunctionCall `json:"functionCall,omitempty"`
    FunctionResp    *FunctionResp `json:"functionResponse,omitempty"`
}

// FunctionCall represents a function call from the model
type FunctionCall struct {
    Name             string         `json:"name"`
    Args             map[string]any `json:"args"`
    ThoughtSignature string         `json:"thoughtSignature,omitempty"`  // ADD THIS (if present here)
}
```

### 2. `/Users/justinjones/Developer/temple-square/app/internal/agent/agent.go`

In the tool calling loop, preserve thought_signature:

```go
// Execute each function call and collect responses
funcRespParts := make([]*Part, 0, len(funcCalls))
for _, fc := range funcCalls {
    // ... existing tool invocation code ...

    // Include thought_signature if present
    part := &Part{
        FunctionResp: &FunctionResp{
            Name:     fc.Name,
            Response: map[string]any{"result": result},
        },
    }

    // Preserve thought signature from function call
    if fc.ThoughtSignature != "" {
        part.ThoughtSignature = fc.ThoughtSignature
    }

    funcRespParts = append(funcRespParts, part)
}
```

Also need to extract thought_signature when parsing function calls from response.

## Deployment
```bash
cd /Users/justinjones/Developer/temple-square/app
go build ./...
gcloud run deploy prophet-agent --source . --region us-central1 --allow-unauthenticated \
  --set-env-vars="TOOLBOX_URL=https://prophet-toolbox-3izw7vdi5a-uc.a.run.app,HTTP_PORT=8080" \
  --set-secrets="GEMINI_API_KEY=GEMINI_API_KEY:latest"
```

## Validation
- No thought_signature errors
- All 3 sections return data via SSE
- Test: `curl "https://prophet-agent-594677951902.us-central1.run.app/api/stream?q=What+is+faith"`

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-thought-sig-fix-04-report.md`

## Go/No-Go
- Go: All sections work with gemini-3-flash-preview
- No-Go: Report specific blocker
