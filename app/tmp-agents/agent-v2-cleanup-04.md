# L8 Principal Engineer Handoff: Update CLAUDE.md with Local References

## Role
You are an L8 principal engineer documenting authoritative local reference material.

## Goal
Update CLAUDE.md with local repository references that are the gold standard for patterns and implementations.

## Hard Requirements
- Local repos are authoritative - they supersede training data
- Document explicit paths for agents to reference
- Make clear these are maintained by the project owner

## Tasks
1. Read current `/Users/justinjones/Developer/temple-square/CLAUDE.md`
2. Add a new section "## Local Reference Repositories (Authoritative)" near the top
3. Document these local paths as gold standard references:
   - `/Users/justinjones/Developer/agent-references/adk-go` - ADK Go SDK source
   - `/Users/justinjones/Developer/agent-references/genai-toolbox` - GenAI Toolbox source
   - `/Users/justinjones/Developer/agent-references/go-genai` - Go GenAI SDK source
   - `/Users/justinjones/Developer/agent-references/google-cloud-go` - Google Cloud Go SDK
   - `/Users/justinjones/Developer/agent-references/mcp-toolbox-sdk-go` - MCP Toolbox SDK for Go
4. Add explicit instructions that agents should use Explore agents to find patterns in these repos
5. Note that these are kept updated and supersede LLM training data

## Report
Write completion report to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-v2-cleanup-04-report.md`

## Conventional Commit
```
docs(claude): add authoritative local reference repositories
```
