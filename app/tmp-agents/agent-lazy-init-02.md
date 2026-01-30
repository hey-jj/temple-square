# L8 Implementation Prompt: Lazy Initialization for Fast Startup

## Role
L8 Principal Engineer - Implementation Executor

## Goal
Fix Cloud Run startup timeout by implementing lazy initialization. Currently the agent loads MCP Toolbox tools at server startup, which takes too long and fails health checks.

## Problem
Cloud Run health checks fail with "Startup probes timed out after 5m5s" because:
1. `New()` in agent.go loads all toolsets during initialization
2. This blocks server startup
3. Health check timeouts before server is ready

## Solution
Implement lazy initialization:
1. Create toolbox client at startup (fast)
2. Defer toolset loading until first request
3. Use sync.Once to ensure tools are loaded only once

## Hard Requirements
1. Server must start and be healthy within 10 seconds
2. Tools are loaded on first request, not at startup
3. Thread-safe lazy initialization using sync.Once
4. Keep using gemini-3-flash-preview model
5. No other behavioral changes

## Files to Modify
- `/Users/justinjones/Developer/temple-square/app/internal/agent/agent.go`

## Implementation Pattern
```go
type ProphetAgent struct {
    client         *GeminiClient
    toolboxClient  *core.ToolboxClient
    toolboxURL     string

    // Lazy-loaded tools
    initOnce        sync.Once
    initErr         error
    presidentsTools []*core.ToolboxTool
    leadersTools    []*core.ToolboxTool
    scripturesTools []*core.ToolboxTool
}

func New(ctx context.Context, cfg Config) (*ProphetAgent, error) {
    // Create Gemini client (fast)
    client, err := NewGeminiClient(cfg.APIKey)
    if err != nil {
        return nil, err
    }

    // Just store the URL, don't connect yet
    toolboxURL := cfg.ToolboxURL
    if toolboxURL == "" {
        toolboxURL = os.Getenv("TOOLBOX_URL")
    }

    return &ProphetAgent{
        client:     client,
        toolboxURL: toolboxURL,
    }, nil
}

func (a *ProphetAgent) ensureInitialized(ctx context.Context) error {
    a.initOnce.Do(func() {
        // Load toolbox and tools here
        toolboxClient, err := core.NewToolboxClient(a.toolboxURL)
        if err != nil {
            a.initErr = err
            return
        }
        a.toolboxClient = toolboxClient

        // Load toolsets...
    })
    return a.initErr
}

func (a *ProphetAgent) Run(ctx context.Context, question string) <-chan AgentResult {
    results := make(chan AgentResult, 3)

    if err := a.ensureInitialized(ctx); err != nil {
        go func() {
            results <- AgentResult{Error: err}
            close(results)
        }()
        return results
    }

    // ... rest of Run implementation
}
```

## Tasks
1. Modify New() to not load tools at startup
2. Add ensureInitialized() method with sync.Once
3. Call ensureInitialized() at start of Run()
4. Build and test locally
5. Deploy to Cloud Run
6. Verify startup is fast and first request works

## Validation
- Server starts within 10 seconds
- First request loads tools (may be slow)
- Subsequent requests are fast
- All 3 sections return data

## Report
Write your report to:
`/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-lazy-init-02-report.md`

## Go/No-Go
- Go: Deployed, healthy, all sections work
- No-Go: Report specific blocker
