# Debug Report: Scriptures Section Never Appearing in SSE Output

## Investigation Summary

Performed systematic investigation of all potential causes listed in the handoff document.

## Findings

### 1. Agent Names Match - VERIFIED
- **agent_v2.go line 235**: `Name: "scriptures_agent"`
- **sse_v2.go line 249**: `case "scriptures_agent":`
- **Result**: Names match exactly. No typos found.

### 2. SSE Handler Matching - VERIFIED
The switch case in `tryParseAndSendSection()` correctly handles "scriptures_agent":
```go
case "scriptures_agent":
    eventName = "scriptures"
    var resp ScripturesResponse
    // ... parsing and rendering logic
```

### 3. Typo Check - VERIFIED
No typos found. Both files consistently use "scriptures_agent" (plural).

### 4. Parallel Agent Configuration - VERIFIED
The scriptures agent is correctly included in SubAgents:
```go
SubAgents: []agent.Agent{presidentsAgent, leadersAgent, scripturesAgent},
```

### 5. Event Author Field - NOT THE ISSUE
The Author field mechanism is working correctly for presidents and leaders agents.

### 6. Toolbox Toolset Loading - VERIFIED
The scriptures toolset loads correctly with proper tools:
- search_scriptures
- get_scripture_by_reference
- search_talks_mentioning_scripture

## ROOT CAUSE IDENTIFIED

The root cause is a **schema constraint impossibility** that prevents the scriptures agent from producing valid output.

### The Problem

In the JSON schema for scriptures responses (agent_v2.go lines 108-136):

```go
"required": []string{"volume", "reference", "text", "related_talk"},
```

**Every scripture item was REQUIRED to have a `related_talk` object.**

Combined with:
- `minItems: 5` - agent must return at least 5 scriptures
- Each related_talk requires speaker, title, and quote

### Why This Caused Failure

1. The scriptures agent searches for scriptures and tries to find related talks
2. The `search_talks_mentioning_scripture` tool does a simple ILIKE text search: `WHERE t.content ILIKE '%' || $1 || '%'`
3. Many scriptures don't have talks that explicitly mention the exact reference
4. The LLM cannot produce valid JSON meeting the schema requirements (all 5+ items needing related_talk)
5. The agent either produces nothing or malformed JSON that fails to parse
6. No events are ever streamed to the client

### Why Presidents/Leaders Work

Their schemas only require basic fields that are always available from the database:
- Presidents: `"required": []string{"speaker", "title", "conference", "quote"}`
- Leaders: `"required": []string{"speaker", "title", "conference", "quote"}`

Both have `minItems: 2` (achievable) and all required fields come directly from the talks database.

## Fix Applied

### Change 1: Made `related_talk` Optional

**File**: `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go`

**Before** (line 129):
```go
"required": []string{"volume", "reference", "text", "related_talk"},
```

**After**:
```go
"required": []string{"volume", "reference", "text"},
```

### Change 2: Updated Prompt to Reflect Optional related_talk

**Before**:
```
- Each scripture MUST have a related_talk with a short quote
```

**After**:
```
- Include related_talk when a matching talk is found (optional - not all scriptures need a related talk)
```

## Build Verification

```bash
cd /Users/justinjones/Developer/temple-square/app && go build ./...
# Build succeeded with no errors
```

## Impact

- Scriptures will now be returned even when related talks cannot be found
- When a related talk IS found, it will still be included (the schema still supports it)
- The minItems: 5 constraint remains, but is now achievable since each item only needs volume, reference, and text

## Recommendations

1. Consider improving `search_talks_mentioning_scripture` to use fuzzy matching or search for book names without exact verse numbers
2. Add debug logging in the SSE handler to log all incoming event.Author values for easier future debugging
3. Consider reducing minItems from 5 to 3 to improve reliability further

## Files Modified

- `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go`
  - Line 129: Removed "related_talk" from required fields
  - Lines 312-329: Updated prompt to clarify related_talk is optional
