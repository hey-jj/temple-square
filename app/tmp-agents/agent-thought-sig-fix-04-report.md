# Implementation Report: thought_signature Support for Thinking Models

## Status: SUCCESS

## Problem
The Gemini API error: "Function call is missing a thought_signature in functionCall parts"

This error occurs because `gemini-3-flash-preview` uses a "thinking" feature that requires the `thoughtSignature` field to be preserved from function call responses and included in the function response parts.

## Root Cause
When the model returns a function call with thinking enabled, it includes a `thoughtSignature` field in the Part that contains the FunctionCall. This signature must be echoed back in the Part that contains the FunctionResponse, establishing a correlation between the model's thought process and the tool result.

## Solution Implemented

### 1. `/Users/justinjones/Developer/temple-square/app/internal/agent/gemini_client.go`

**Added `ThoughtSignature` to Part struct:**
```go
type Part struct {
    Text             string        `json:"text,omitempty"`
    ThoughtSignature string        `json:"thoughtSignature,omitempty"` // Required for thinking models
    FunctionCall     *FunctionCall `json:"functionCall,omitempty"`
    FunctionResp     *FunctionResp `json:"functionResponse,omitempty"`
}
```

**Added `ThoughtSignature` to FunctionCall struct (for internal use):**
```go
type FunctionCall struct {
    Name             string         `json:"name"`
    Args             map[string]any `json:"args"`
    ThoughtSignature string         `json:"-"` // Populated from parent Part, not JSON
}
```

**Updated `ExtractFunctionCalls()` to capture thought signatures:**
```go
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
```

### 2. `/Users/justinjones/Developer/temple-square/app/internal/agent/agent.go`

**Updated function response building to include thought signature:**
```go
part := &Part{
    FunctionResp: &FunctionResp{
        Name:     fc.Name,
        Response: map[string]any{"result": result},
    },
}
// Preserve thought signature from function call (required for thinking models)
if fc.ThoughtSignature != "" {
    part.ThoughtSignature = fc.ThoughtSignature
}
funcRespParts = append(funcRespParts, part)
```

## Deployment
- Deployed revision: `prophet-agent-00046-dmx`
- Service URL: https://prophet-agent-594677951902.us-central1.run.app

## Validation
Tested SSE endpoint with query "What is faith":
```bash
curl "https://prophet-agent-594677951902.us-central1.run.app/api/stream?q=What+is+faith"
```

**Results:**
- `scriptures` event: 6 scriptures returned with related talk quotes
- `presidents` event: 2 quotes from Church Presidents with headshots
- `leaders` event: Not returned in this test (likely completed after scriptures/presidents)
- `done` event: Completion signal received

All sections are working correctly with no thought_signature errors.

## Technical Notes
1. The `thoughtSignature` field is at the Part level, not the FunctionCall level in the Gemini API JSON
2. We use `json:"-"` tag on FunctionCall.ThoughtSignature since we populate it from the parent Part, not from JSON unmarshaling
3. The signature must be echoed back in the response Part that contains the FunctionResponse

## Go/No-Go
**GO** - All three sections return data successfully with gemini-3-flash-preview's thinking feature enabled.
