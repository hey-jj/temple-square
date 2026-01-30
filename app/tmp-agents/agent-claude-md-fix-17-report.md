# Agent Report: CLAUDE.md Fix and V2 Reference Audit

## Task Summary
1. Add "Pinned Defaults (MANDATORY)" section to top of CLAUDE.md
2. Search codebase for "v2" references that should be removed

## Actions Completed

### 1. CLAUDE.md Updated

Added new mandatory section at the VERY TOP of `/Users/justinjones/Developer/temple-square/CLAUDE.md`:

```markdown
## Pinned Defaults (MANDATORY)

**THIS SECTION IS NON-NEGOTIABLE. ALL AGENTS MUST FOLLOW THESE DEFAULTS.**

| Setting | Value | Notes |
|---------|-------|-------|
| **Model** | `gemini-3.0-flash-preview` | NEVER use 2.0, 2.5, or any other version. Always use 3.0-flash-preview. |
| **Reference** | https://aistackregistry.com | Authoritative source for model defaults and configurations |

**CRITICAL RULES:**
1. When specifying Gemini models, ALWAYS use `gemini-3.0-flash-preview`
2. Do NOT substitute with `gemini-2.0-flash`, `gemini-2.5-flash`, or any other variant
3. If you see code using older model versions, flag it for update
4. Reference https://aistackregistry.com for the latest approved model configurations
```

### 2. V2 Reference Audit

#### Files with "v2" in Filename (Project Code - Require Review)

| File | Location | Recommendation |
|------|----------|----------------|
| `agent_v2.go` | `/Users/justinjones/Developer/temple-square/app/internal/agent/agent_v2.go` | **REVIEW** - Core agent implementation |
| `sse_v2.go` | `/Users/justinjones/Developer/temple-square/app/cmd/server/sse_v2.go` | **REVIEW** - SSE handler |
| `parse_2024a_talks_v2.py` | `/Users/justinjones/Developer/temple-square/tmp/parse_2024a_talks_v2.py` | In tmp folder, likely scratch file |

#### Agent Report Files (tmp-agents - Cleanup Candidates)

These are historical agent reports and can be cleaned up:

- `agent-v2-cleanup-01.md` through `agent-v2-cleanup-07.md`
- `agent-v2-cleanup-01-report.md` through `agent-v2-cleanup-07-report.md`
- `v2-architecture-complete-summary.md`

#### CLAUDE.md References to V2

The CLAUDE.md file itself contains a section titled "## V2 Architecture (Current)" that references:
- `USE_V2=true` environment variable
- `agent_v2.go` and `sse_v2.go` files
- `make dev-v2` command

#### Third-Party/Node Modules (Ignore)

158 total files matched, but most are in `node_modules` directories (playwright, undici, types, etc.) - these are third-party dependencies and should be ignored.

## Recommendations

### High Priority
1. **Rename core v2 files** - If v2 is now the standard architecture:
   - `agent_v2.go` -> `agent.go` (remove old v1 if exists)
   - `sse_v2.go` -> `sse.go` (remove old v1 if exists)

2. **Update CLAUDE.md** - Rename "V2 Architecture (Current)" section to just "Architecture" since v2 is now the standard

3. **Update environment variables** - Consider removing `USE_V2=true` flag if v1 is deprecated

### Low Priority
4. **Clean tmp-agents** - Remove old v2 cleanup reports once migration is confirmed complete

5. **Clean tmp folder** - Review `parse_2024a_talks_v2.py` for deletion

## Files Modified
- `/Users/justinjones/Developer/temple-square/CLAUDE.md` - Added Pinned Defaults section at top

## Status: COMPLETE
