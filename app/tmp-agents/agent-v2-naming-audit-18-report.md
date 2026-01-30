# V2 Naming Artifacts Audit Report

**Date:** 2026-01-28
**Scope:** Full codebase search for "v2" and "V2" naming artifacts

---

## Summary

Found **4 source files** containing V2 naming artifacts that need to be cleaned up.

---

## 1. Files to Rename

| Current File | Should Be Renamed To |
|-------------|---------------------|
| `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go` | `agent.go` |
| `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go` | `sse.go` |

---

## 2. Function/Type Names to Rename

### `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go`

| Line | Current Name | Should Be |
|------|-------------|-----------|
| 138-139 | `ConfigV2` (struct) | `Config` |
| 145-146 | `NewV2` (function) | `New` |
| 280 | `presidentsPromptV2` (const) | `presidentsPrompt` |
| 297 | `leadersPromptV2` (const) | `leadersPrompt` |
| 314 | `scripturesPromptV2` (const) | `scripturesPrompt` |

**References to update within this file:**
- Line 202: `Instruction: presidentsPromptV2` -> `Instruction: presidentsPrompt`
- Line 221: `Instruction: leadersPromptV2` -> `Instruction: leadersPrompt`
- Line 240: `Instruction: scripturesPromptV2` -> `Instruction: scripturesPrompt`

### `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go`

| Line | Current Name | Should Be |
|------|-------------|-----------|
| 62-63 | `handleSSEStreamV2` (function) | `handleSSEStream` |

### `/Users/justinjones/Developer/temple-square/app/internal/ui/components/sections.templ`

| Line | Current Name | Should Be |
|------|-------------|-----------|
| 165-166 | `ScriptureCardV2` (templ) | `ScriptureCard` (see note below) |
| 205-206 | `ScripturesSectionV2` (templ) | `ScripturesSection` (see note below) |
| 210 | `@ScriptureCardV2(scripture)` | `@ScriptureCard(scripture)` |

**Note:** The existing `ScriptureCard` and `ScripturesSection` functions exist but use the old `ScriptureRef` type. The V2 versions use `ScriptureWithTalk` type. Consider:
- Option A: Remove the old versions and rename V2 to standard names
- Option B: Keep both with different names (but remove V2 suffix)

---

## 3. Comments to Update

### `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go`

| Line | Current Comment | Should Be |
|------|----------------|-----------|
| 1 | `// cmd/server/sse_v2.go` (in sse_v2.go header) | Remove filename comment or update to `sse.go` |
| 138 | `// ConfigV2 holds agent configuration for v2 architecture` | `// Config holds agent configuration` |
| 145 | `// NewV2 creates a new prophet agent using parallel workflow agents with MCP Toolbox` | `// New creates a new prophet agent using parallel workflow agents with MCP Toolbox` |
| 266 | `log.Println("Prophet agent v2 initialized with parallel workflow and MCP Toolbox")` | `log.Println("Prophet agent initialized with parallel workflow and MCP Toolbox")` |
| 279 | `// Prompts for v2 agents with structured output focus` | `// Prompts for agents with structured output focus` |

### `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go`

| Line | Current Comment | Should Be |
|------|----------------|-----------|
| 1 | `// cmd/server/sse_v2.go` | `// cmd/server/sse.go` |
| 2 | `// SSE handler for v2 parallel agent architecture with structured outputs` | `// SSE handler for parallel agent architecture with structured outputs` |
| 62 | `// handleSSEStreamV2 handles SSE streaming for v2 parallel agent architecture` | `// handleSSEStream handles SSE streaming for parallel agent architecture` |

### `/Users/justinjones/Developer/temple-square/app/cmd/server/main.go`

| Line | Current Comment/Code | Should Be |
|------|---------------------|-----------|
| 172 | `// V2: Parallel agent architecture with MCP Toolbox (required)` | `// Parallel agent architecture with MCP Toolbox (required)` |
| 178 | `log.Println("Starting with v2 architecture (parallel agents with MCP Toolbox)")` | `log.Println("Starting with parallel agents and MCP Toolbox")` |

---

## 4. References That Need Updating After Renames

### `/Users/justinjones/Developer/temple-square/app/cmd/server/main.go`

| Line | Current Reference | After Rename |
|------|------------------|--------------|
| 179 | `prophetagent.NewV2(ctx, prophetagent.ConfigV2{` | `prophetagent.New(ctx, prophetagent.Config{` |
| 279 | `handleSSEStreamV2(w, r, adkRunner, sessionService)` | `handleSSEStream(w, r, adkRunner, sessionService)` |

### `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go` (will be sse.go)

| Line | Current Reference | After Rename |
|------|------------------|--------------|
| 263 | `components.ScripturesSectionV2(scriptures)` | `components.ScripturesSection(scriptures)` |

### `/Users/justinjones/Developer/temple-square/app/internal/ui/components/sections_templ.go` (auto-generated)

This file is auto-generated from `sections.templ`. After updating `sections.templ` and running `templ generate`, this file will be regenerated automatically. No manual edits needed.

| Line | Current | Will Become |
|------|---------|-------------|
| 533-534 | `func ScriptureCardV2` | `func ScriptureCard` |
| 651-652 | `func ScripturesSectionV2` | `func ScripturesSection` |
| 682 | `ScriptureCardV2(scripture)` | `ScriptureCard(scripture)` |

---

## 5. Files NOT Requiring Changes

The following files contain "V2" but are external dependencies in `go.sum` - these are version numbers in dependency URLs and should NOT be changed:

- `go.sum` - Contains `v2` in module version paths (e.g., `modernc.org/cc/v4`) - these are normal Go module versioning

---

## 6. Execution Order Recommendation

1. **Rename files first:**
   - `agent_v2.go` -> `agent.go`
   - `sse_v2.go` -> `sse.go`

2. **Update function/type names in renamed files:**
   - In `agent.go`: `ConfigV2` -> `Config`, `NewV2` -> `New`, prompt consts
   - In `sse.go`: `handleSSEStreamV2` -> `handleSSEStream`

3. **Update sections.templ:**
   - `ScriptureCardV2` -> rename (handle conflict with existing `ScriptureCard`)
   - `ScripturesSectionV2` -> rename (handle conflict with existing `ScripturesSection`)

4. **Update main.go references:**
   - Update import usage of `NewV2` -> `New`, `ConfigV2` -> `Config`
   - Update function call `handleSSEStreamV2` -> `handleSSEStream`

5. **Update sse.go reference:**
   - Update `components.ScripturesSectionV2` -> `components.ScripturesSection`

6. **Regenerate templ:**
   - Run `templ generate` to update `sections_templ.go`

7. **Verify build:**
   - Run `go build ./...` to ensure no broken references

---

## 7. Total Count of V2 Artifacts

| Category | Count |
|----------|-------|
| Files to rename | 2 |
| Types/Structs to rename | 1 (`ConfigV2`) |
| Functions to rename | 2 (`NewV2`, `handleSSEStreamV2`) |
| Constants to rename | 3 (`presidentsPromptV2`, `leadersPromptV2`, `scripturesPromptV2`) |
| Templ components to rename | 2 (`ScriptureCardV2`, `ScripturesSectionV2`) |
| Comments to update | 9 |
| Code references to update | 8 |
| **Total artifacts** | **27** |

---

**Report generated by audit task. No changes have been made.**
