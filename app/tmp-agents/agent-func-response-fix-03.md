# L8 Implementation Prompt: Fix Function Response Format for Gemini 3

## Role
L8 Principal Engineer - Implementation Executor

## Goal
Fix the Gemini API 400 error: "Invalid value at 'contents[2].parts[0].function_response.response' (type.googleapis.com/google.protobuf.Struct)"

## Problem
gemini-3-flash-preview requires function responses to be protobuf Struct objects. When tools return arrays, the response is invalid.

Error:
```
Invalid value at 'contents[2].parts[0].function_response.response' (type.googleapis.com/google.protobuf.Struct), "[{\"id\":283,...}]"
```

## Solution
Wrap all function responses in an object with a `result` field:

```go
// Before (fails with array results):
FunctionResp: &FunctionResp{
    Name:     fc.Name,
    Response: result,  // might be array, fails
}

// After (always valid object):
FunctionResp: &FunctionResp{
    Name:     fc.Name,
    Response: map[string]any{"result": result},  // always object
}
```

## Files to Modify
- `/Users/justinjones/Developer/temple-square/app/internal/agent/agent.go`

Lines ~360-380 in runSubAgent() - wrap tool results in object.

## Implementation
1. Find the funcRespParts loop in runSubAgent()
2. Change `Response: result` to `Response: map[string]any{"result": result}`
3. Also wrap error responses: `Response: map[string]any{"error": err.Error()}`
4. Build and deploy

## Deployment
```bash
cd /Users/justinjones/Developer/temple-square/app
go build ./...
gcloud run deploy prophet-agent --source . --region us-central1 --allow-unauthenticated \
  --set-env-vars="TOOLBOX_URL=https://prophet-toolbox-3izw7vdi5a-uc.a.run.app,HTTP_PORT=8080" \
  --set-secrets="GEMINI_API_KEY=GEMINI_API_KEY:latest"
```

## Validation
- No 400 errors in logs
- All 3 sections return data via SSE
- Test: `curl "https://prophet-agent-594677951902.us-central1.run.app/api/stream?q=What+is+faith"`

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-func-response-fix-03-report.md`

## Go/No-Go
- Go: All sections work with gemini-3-flash-preview
- No-Go: Report specific blocker
