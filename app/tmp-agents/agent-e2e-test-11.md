# L8 Principal Engineer Handoff: End-to-End Testing

## Role
You are an L8 principal engineer performing end-to-end testing of the deployed system.

## Goal
Verify all three sections (Presidents, Leaders, Scriptures) are populated with actual content from the SSE stream.

## Context
- Main App URL: https://prophet-agent-594677951902.us-central1.run.app
- Toolbox URL: https://prophet-toolbox-594677951902.us-central1.run.app

## Tasks

### 1. Test SSE Stream Directly
Connect to the SSE endpoint and capture events:
```bash
curl -N "https://prophet-agent-594677951902.us-central1.run.app/api/stream?session=test-$(date +%s)&q=What%20is%20faith%3F" 2>&1 | head -500
```

Look for:
- `event: presidents` with HTML content
- `event: leaders` with HTML content
- `event: scriptures` with HTML content (minimum 5 scriptures)
- `event: done`
- NO `event: error` events

### 2. Verify Presidents Section Content
Check that presidents section contains:
- Speaker names (President Oaks, President Nelson, etc.)
- Headshot images
- Talk titles
- Actual quote text (not placeholders)

### 3. Verify Leaders Section Content
Check that leaders section contains:
- Speaker names (Elder/Sister titles)
- Headshot images
- Talk titles
- Actual quote text

### 4. Verify Scriptures Section Content
Check that scriptures section contains:
- At least 5 scripture references
- Volume labels (New Testament, Book of Mormon, etc.)
- Actual verse text
- Related talk quotes for each scripture

### 5. Test Error Handling
Test with a potentially problematic query:
```bash
curl -N "https://prophet-agent-594677951902.us-central1.run.app/api/stream?session=test-err-$(date +%s)&q=Tell%20me%20about%20controversial%20church%20history" 2>&1 | head -100
```

Should receive a redirect response, not crash.

### 6. Check Cloud Run Logs for Errors
```bash
gcloud run services logs read prophet-agent --region=us-central1 --project=temple-square --limit=50 2>&1 | grep -i "error\|ERROR\|panic\|failed" | head -20
```

## Report
Write to: `/Users/justinjones/Developer/temple-square/app/tmp-agents/agent-e2e-test-11-report.md`

Include:
- SSE events received (list event types)
- Content verification for each section (PASS/FAIL)
- Any errors in logs
- Overall Go/No-Go

## Go/No-Go Criteria
- **GO**: All three sections populated with real content, no errors
- **NO-GO**: Any section empty, error events, or crashes
