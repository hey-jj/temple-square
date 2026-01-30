# L8 Principal Engineer Handoff: Verify Scriptures Implementation

## Role
You are an L8 principal engineer verifying the scriptures agent meets requirements.

## Goal
Verify the scriptures agent implementation enforces minimum 5 outputs with related talk quotes.

## Requirements to Verify
1. Scripture schema enforces minItems: 5
2. Each scripture MUST have related_talk field (required, not optional)
3. Related talk has speaker, title, quote fields
4. Tools are configured to support finding talks that reference scriptures

## Tasks

### 1. Verify Schema in agent_v2.go
Read `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go`
Check `scripturesSchema`:
- [ ] Has `"minItems": 5` (or higher)
- [ ] `related_talk` is in the required array for each scripture item
- [ ] related_talk object has speaker, title, quote as required

### 2. Verify Tools Configuration
Read `/Users/justinjones/Developer/temple-square/app/tools.yaml`
Check scriptures toolset has:
- [ ] search_scriptures tool
- [ ] search_talks_mentioning_scripture tool (critical for finding related talks)

### 3. Verify SSE Handler
Read `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go`
Check:
- [ ] ScripturesResponse struct has Scriptures field
- [ ] StructuredScripture has RelatedTalk field
- [ ] convertStructuredScriptures handles RelatedTalk correctly

### 4. Verify UI Component
Read `/Users/justinjones/Developer/temple-square/app/internal/ui/components/sections.templ`
Check:
- [ ] ScriptureWithTalk type exists with RelatedTalk field
- [ ] ScriptureCardV2 renders the related talk quote
- [ ] Different styling for related talk (smaller, no headshot)

### 5. Compare with Local Reference
Explore `/Users/justinjones/Developer/agent-references/adk-go/examples/` for structured output patterns
Verify our implementation aligns with ADK best practices

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-scriptures-verify-08-report.md`

Include:
- Each verification result (PASS/FAIL)
- If FAIL, what needs to be fixed
- Specific line numbers for any issues found

## Go/No-Go
- **GO**: All verifications pass
- **NO-GO**: Any verification fails - list specific fixes needed
