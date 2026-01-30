# L8 Principal Engineer Handoff: Remove V1 Agent Code

## Role
You are an L8 principal engineer removing dead v1 code paths.

## Goal
Remove all v1 agent code that is no longer used after v2 cutover.

## Hard Requirements
- Delete dead code completely - no archiving, no commenting
- Fail-fast principle - if something depends on v1, compilation will fail (which is correct)
- No backwards compatibility

## Tasks
1. Delete `/Users/justinjones/Developer/temple-square/app/internal/agent/agent.go` entirely (v1 agent)
2. Read `/Users/justinjones/Developer/temple-square/app/internal/agent/tools.go`
3. If tools.go contains only v1 code (DBQuerier, createTools, createToolsSeparate), delete the entire file
4. If tools.go contains shared code needed by v2, keep only that and remove v1 functions
5. Check for any other files in internal/agent/ that are v1-only and delete them
6. Run `go build ./...` to verify nothing breaks

## Validation
- `go build ./...` must succeed
- No `func New(` signature in internal/agent/ (v1 constructor)
- No DBQuerier interface remains
- No createTools or createToolsSeparate functions remain

## Report
Write completion report to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-v2-cleanup-05-report.md`

## Conventional Commit
```
refactor(agent): remove v1 agent code

BREAKING CHANGE: v1 agent removed, only v2 parallel architecture remains
```
