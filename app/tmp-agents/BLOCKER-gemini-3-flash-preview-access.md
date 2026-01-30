# RESOLVED: gemini-3-flash-preview Model Access

## Issue (RESOLVED)
The model `gemini-3-flash-preview` was returning 404 errors when accessed via Vertex AI.

## Root Cause
The model is accessed via **Gemini API** (`generativelanguage.googleapis.com`), NOT Vertex AI.

## Solution Applied
1. Changed backend from `genai.BackendVertexAI` to `genai.BackendGeminiAPI`
2. Added `GEMINI_API_KEY` requirement (stored in Google Secret Manager as `gemini-api-key`)
3. Updated deployment to inject secret: `--set-secrets="GEMINI_API_KEY=gemini-api-key:latest"`

## Model Specification
From https://aistackregistry.com/latest/models/gemini/gemini-3-flash-preview/spec.json:

```json
{
  "model_id": "gemini-3-flash-preview",
  "provider": "google",
  "sources": [
    "https://generativelanguage.googleapis.com/v1beta/models",
    "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview"
  ],
  "token_limits": {
    "input_token_limit": 1048576,
    "output_token_limit": 65536
  }
}
```

## Files Changed
- `internal/agent/agent.go` - Use BackendGeminiAPI with APIKey
- `Makefile` - Add --set-secrets for GEMINI_API_KEY
- `CLAUDE.md` - Document full spec and secret configuration
