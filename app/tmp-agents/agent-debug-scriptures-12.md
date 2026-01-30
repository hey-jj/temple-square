# L8 Principal Engineer Handoff: Debug Scriptures Section

## Role
You are an L8 principal engineer debugging why scriptures section never appears in SSE output.

## Context
- Presidents section: Works (logs show "Streamed presidents section")
- Leaders section: Works (logs show "Streamed leaders section")
- Scriptures section: NEVER appears in logs - completely missing

## Goal
Find and fix the bug preventing scriptures from being streamed.

## Investigation Tasks

### 1. Check Agent Names Match
Read `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go`
Find the exact `Name` field for the scriptures agent:
```go
llmagent.Config{
    Name: "scriptures_agent",  // Is this exactly what's defined?
```

### 2. Check SSE Handler Matching
Read `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go`
Find the switch case for scriptures:
```go
case "scriptures_agent":  // Does this match exactly?
```

### 3. Check for Typos
Common issues:
- "scripture_agent" vs "scriptures_agent" (singular vs plural)
- Case sensitivity
- Trailing spaces

### 4. Check Parallel Agent Configuration
In agent_v2.go, verify scriptures agent is included in SubAgents:
```go
SubAgents: []agent.Agent{presidentsAgent, leadersAgent, scripturesAgent},
```

### 5. Check Event Author Field
The SSE handler uses `event.Author` to determine which agent produced the event.
Verify this is populated correctly for parallel agents.

### 6. Check Toolbox Toolset Loading
Verify scriptures toolset loads correctly:
```go
scripturesTools, err := toolboxClient.LoadToolset("scriptures", ctx)
```

### 7. Fix the Issue
Once identified, fix the bug.

### 8. Verify Build
```bash
cd /Users/justinjones/Developer/temple-square/app && go build ./...
```

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-debug-scriptures-12-report.md`

Include:
- Root cause identified
- Fix applied
- Build verification

## Critical
This is blocking - scriptures section is a core requirement.
