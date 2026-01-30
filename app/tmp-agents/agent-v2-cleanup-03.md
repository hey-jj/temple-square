# L8 Principal Engineer Handoff: V2 Architecture Validation

## Role
You are an L8 principal engineer performing final validation of the v2 architecture before Go/No-Go decision.

## Goal
Validate that the complete v2 architecture meets all acceptance criteria and is ready for deployment.

## Hard Requirements
- All code must compile without errors
- No v1 code paths remain
- No fallback/backwards compatibility code exists
- All required environment variables cause fail-fast on missing
- Structured outputs are enforced (no text parsing)
- MCP Toolbox integration is properly configured

## Validation Tasks

### 1. Code Compilation
- Run `go build ./cmd/server` and verify success
- Run `go build ./...` to verify all packages compile

### 2. Architecture Review - main.go
Read `/Users/justinjones/Developer/temple-square/app/cmd/server/main.go` and verify:
- [ ] No USE_V2 environment variable
- [ ] No prophetagent.New (v1) references
- [ ] No direct database connection code
- [ ] TOOLBOX_URL is required (fail-fast)
- [ ] Only NewV2 agent creation path exists

### 3. Architecture Review - sse_v2.go
Read `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go` and verify:
- [ ] No extractJSON function
- [ ] No silent return false on errors
- [ ] All JSON parse errors send SSE error events
- [ ] All JSON parse errors are logged with context

### 4. Architecture Review - agent_v2.go
Read `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go` and verify:
- [ ] Uses parallelagent from ADK
- [ ] Uses MCP Toolbox SDK (tbadk)
- [ ] Structured output schemas defined for all agents
- [ ] ResponseMIMEType is "application/json" for all agents
- [ ] ResponseJsonSchema is set for all agents

### 5. Configuration Review - tools.yaml
Read `/Users/justinjones/Developer/temple-square/app/tools.yaml` and verify:
- [ ] Database source is configured
- [ ] All required tools are defined (search_scriptures, get_presidents_talks, etc.)
- [ ] Toolsets are defined (scriptures, presidents, leaders)

### 6. Component Review - sections.templ
Read `/Users/justinjones/Developer/temple-square/app/internal/ui/components/sections.templ` and verify:
- [ ] ScriptureWithTalk type exists
- [ ] TalkPullQuote type exists
- [ ] ScriptureCardV2 template exists
- [ ] ScripturesSectionV2 template exists

### 7. No Dead Code Check
Verify no orphaned v1 functions remain in any files:
- parseSpeakerQuotes
- parseScriptureRefs
- determineVolume
- renderSectionsFromResponse
- handleSSEStream (v1 handler)

## Report
Write your completion report to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-v2-cleanup-03-report.md`

Include:
- Checklist results (pass/fail for each item)
- Any violations found
- Final Go/No-Go recommendation
- If No-Go, list specific items that must be fixed

## Go/No-Go Decision
- **Go**: All checklist items pass, no violations found
- **No-Go**: Any checklist item fails or violation found

Provide explicit Go/No-Go with justification.
