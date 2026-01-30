# L8 Principal Engineer Handoff: Remove Silent Fallbacks in SSE Handler

## Role
You are an L8 principal engineer enforcing fail-fast error handling. No silent recovery, no fallbacks.

## Goal
Remove all silent fallback logic from sse_v2.go. JSON parsing must fail explicitly, not silently recover.

## Hard Requirements
- NO silent JSON extraction fallbacks (remove `extractJSON` function)
- NO silent `return false` on parse errors - must log and surface errors
- FAIL-FAST with explicit error messages to client
- Structured output from agents MUST be valid JSON - if not, it's an agent configuration error
- Surface all errors through SSE error events to the client

## Tasks
1. Read `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go`
2. Remove the `extractJSON` function entirely
3. In `tryParseAndSendSection`, remove the fallback JSON extraction logic
4. When JSON parsing fails, send an SSE error event with the agent name and error details
5. Return explicit errors instead of silent `return false`
6. Log all parsing failures with agent name and content snippet for debugging
7. Ensure the client receives clear feedback when agent output is malformed

## Validation
- `go build ./cmd/server` must succeed
- No `extractJSON` function exists
- No silent `return false` after JSON unmarshal errors
- All parse failures send SSE error events
- All parse failures are logged with context

## Acceptance Criteria
- Fail-fast on malformed JSON (explicit error, not silent skip)
- Client receives error events for parse failures
- Server logs capture agent name + error for debugging
- No dead code or silent recovery paths

## Report
Write your completion report to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-v2-cleanup-02-report.md`

Include:
- Functions removed
- Error handling changes made
- Validation results (build output)
- Any blockers encountered

## Conventional Commit
```
fix(sse): remove silent fallbacks, fail-fast on malformed JSON

Agents must produce valid structured output. Parse failures are
now surfaced as SSE error events and logged for debugging.
```

## Go/No-Go
No-Go if: compilation fails, extractJSON still exists, or silent return false remains
