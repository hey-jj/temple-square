# Concurrency Bug Fix Report: strings.Builder Panic in SSE Handler

## Summary

Fixed a critical bug in `/Users/justinjones/Developer/temple-square/app/cmd/server/sse.go` where `strings.Builder` values were being copied after first use, which can cause panics due to shared internal buffer state.

## The Bug

### Location
File: `/Users/justinjones/Developer/temple-square/app/cmd/server/sse.go`
Lines: 125-152 (original lines 123-149)

### Original Code
```go
agentResponses := make(map[string]strings.Builder)
// ...
for _, part := range event.Content.Parts {
    if part.Text != "" {
        builder := agentResponses[agentName]  // Gets VALUE (copy)
        builder.WriteString(part.Text)        // Writes to copy
        agentResponses[agentName] = builder   // Stores back
    }
}
```

### Root Cause
The Go documentation for `strings.Builder` explicitly states:

> "A Builder must not be copied after first use."

The problem is that `strings.Builder` contains an internal byte slice buffer. When you copy a `Builder` value (not a pointer), both the original and the copy share the same underlying buffer array. Subsequent writes to either can:

1. Cause data corruption
2. Trigger panics when the Builder detects it has been copied (via `copyCheck()`)
3. Result in non-deterministic behavior under concurrent access

### Context
This SSE handler processes events from a parallel agent architecture that runs three sub-agents (`presidents_agent`, `leaders_agent`, `scriptures_agent`) concurrently. While the event processing loop itself is sequential, the repeated copy-write-store pattern for `strings.Builder` values violates the "no copy after first use" constraint.

## The Fix

### Changed Code
```go
// Use pointers to strings.Builder to avoid copying after first write.
// Per Go docs: "A Builder must not be copied after first use."
// Copying a Builder with data can cause panics due to shared internal buffer.
agentResponses := make(map[string]*strings.Builder)
// ...
for _, part := range event.Content.Parts {
    if part.Text != "" {
        // Get or create builder for this agent (use pointer to avoid copy)
        builder, ok := agentResponses[agentName]
        if !ok {
            builder = &strings.Builder{}
            agentResponses[agentName] = builder
        }
        builder.WriteString(part.Text)
    }
}
```

### Why This Fix Works
1. **Pointer semantics**: By storing `*strings.Builder` in the map, we retrieve and work with the same instance rather than a copy
2. **Lazy initialization**: Each builder is created once when first needed and reused for all subsequent writes
3. **No value copies**: The pointer is copied (which is safe), not the Builder struct itself

## Verification

Build verification passed:
```
$ go build ./...
# (no errors)
```

## Risk Assessment

- **Impact**: HIGH - The original bug could cause panics in production under load
- **Fix Risk**: LOW - The change is minimal and uses standard Go patterns
- **Regression Risk**: LOW - The behavior is identical, just with correct memory semantics

## Files Modified

1. `/Users/justinjones/Developer/temple-square/app/cmd/server/sse.go`
   - Changed `agentResponses` from `map[string]strings.Builder` to `map[string]*strings.Builder`
   - Added explicit initialization check before first write
   - Added documentation comments explaining the issue

## Additional Notes

While the event loop is sequential (range over iterator), the underlying parallel agent sends events from multiple goroutines into the channel. If the ADK's iterator implementation changes in the future to be more concurrent, this fix ensures the code remains safe.
