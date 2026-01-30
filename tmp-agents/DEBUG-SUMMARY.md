# Debug Mission Report: scriptures.byu.edu Extraction Failure

**Date:** 2026-01-27
**URL Tested:** https://scriptures.byu.edu/#:t210c:g8be
**Status:** ✅ SUCCESS - Root cause identified and solution implemented

---

## Executive Summary

The extraction was failing because scriptures.byu.edu is a **JavaScript Single Page Application (SPA)** that loads content dynamically via AJAX after the initial page load. Static HTTP requests (like WebFetch or curl) only retrieve an HTML shell with "Loading..." placeholders, not the actual talk content.

**Solution Found:** Discovered the internal AJAX API endpoint that the JavaScript uses to fetch talk content. We can now extract content directly from this API without needing browser automation.

---

## Investigation Process

### 1. MCP Tools Availability Check

**Requested:** Chrome DevTools MCP for browser automation
**Result:** Not available in current Claude Code session

**Available Tools:**
- Bash, Glob, Grep, Read, Edit, Write
- WebFetch, WebSearch
- Skill, Task tools

**Finding:** Chrome DevTools MCP is configured in `.mcp.json` but not exposed to this session. The MCP server needs to be running and connected.

**Verdict:** Not a blocker - found a better solution using the AJAX API directly.

---

### 2. Static HTTP Request Test

**Method:** Python requests + BeautifulSoup
**URL:** https://scriptures.byu.edu/#:t210c:g8be

**Results:**
- ✅ HTTP 200 OK
- ✅ Page loads successfully
- ❌ Body contains only: "Loading...\\nLoading..."
- ❌ 0 paragraphs found
- ❌ No talk title
- ❌ No speaker information
- ❌ No content

**HTML Structure:**
```html
<body>
    <div id="scriptures" class="sidecolumn">
        <img src="/static/homepage/media/images/loading.gif" />
        Loading...
        <img src="/static/homepage/media/images/loading.gif" />
    </div>
    <div id="centercolumn"></div>
    <div id="citationindex" class="sidecolumn">
        <img src="/static/homepage/media/images/loading.gif" />
        Loading...
    </div>
</body>
```

**Key Finding:** Content divs are empty on initial load. JavaScript populates them after page renders.

---

### 3. JavaScript Analysis

**File Examined:** https://scriptures.byu.edu/static/homepage/scripts/base.js (3,413 lines)

**Key Functions Discovered:**

#### `navigate()` - Main navigation handler
```javascript
if (changedPieces.center !== undefined) {
    center = decodeCenter(changedPieces.center);
    url = baseUrl + center;
    type = 'talk';
    pendingRequest = $.get(url)
                          .success(navigators[type].successCallback)
                          .error(navigators[type].failureCallback);
}
```

#### `decodeCenter()` - Decodes the hash fragment
```javascript
decodeCenter = function (encodedCenter) {
    var result, type;
    type = encodedCenter.substr(0,1);
    switch(type) {
    case 't':
        // Talk
        result = decodeTalk(encodedCenter.substr(1));
        if (result[0] != 'about'){
            result = '/content/talks_ajax/' + result.join('/');
        }
        break;
    }
    return result;
}
```

#### `decodeTalk()` - Parses talk identifier
```javascript
decodeTalk = function (encodedTalk) {
    var talkid, target='', query='', result;
    encodedTalk = encodedTalk.split('&')
    if (encodedTalk.length > 1) {
        query = encodedTalk[1]
    }
    encodedTalk = encodedTalk[0].split('$')
    if (encodedTalk.length > 1) {
        target = encodedTalk[1];
    }
    talkid = parseInt(encodedTalk[0], 16);
    result = [talkid, query];
    return result;
}
```

**AJAX Endpoint Pattern Found:**
```
/content/talks_ajax/{talk_id_decimal}/{query}
```

---

### 4. Hash Decoding Algorithm

**Original Hash:** `#:t210c:g8be`

**Decoding Steps:**

1. **Remove prefix:** `#:` → Result: `t210c:g8be`
2. **Extract type:** First char `t` → Talk type
3. **Extract encoded value:** `210c:g8be`
4. **Split by `&`:** Check for query params → None found
5. **Split by `$`:** Check for target anchor → None found
6. **Extract conference ID:** Take part before `:` → `210c`
7. **Convert hex to decimal:** `parseInt('210c', 16)` → **8460**
8. **Build API URL:** `https://scriptures.byu.edu/content/talks_ajax/8460`

**Python Implementation:**
```python
conf_hex = "210c"
talk_id_decimal = int(conf_hex, 16)  # Result: 8460
api_url = f"https://scriptures.byu.edu/content/talks_ajax/{talk_id_decimal}"
```

---

### 5. API Endpoint Verification

**Test URL:** https://scriptures.byu.edu/content/talks_ajax/8460

**Results:**
- ✅ HTTP 200 OK
- ✅ Content-Type: text/html; charset=utf-8
- ✅ Content Length: 6,889 bytes
- ✅ Contains full talk content

**Extracted Data:**
```json
{
  "conference": "October 2020, Session 6",
  "speaker": "President Russell M. Nelson",
  "calling": "President of The Church of Jesus Christ of Latter-day Saints",
  "title": "Moving Forward",
  "kicker": "The work of the Lord is steadily moving forward.",
  "paragraphs": 12,
  "scripture_references": [
    "Doctrine and Covenants 84:88",
    "Helaman 3:29",
    "2 Nephi 1:15"
  ]
}
```

---

## Root Cause Analysis

### Why Extraction Was Failing

1. **Client-Side Routing:** URL uses hash fragment (`#:t210c:g8be`) for navigation
2. **JavaScript Execution Required:** Content loaded via AJAX after page renders
3. **Empty Initial HTML:** Server returns shell with "Loading..." placeholders
4. **Static Requests Insufficient:** Tools like WebFetch can't execute JavaScript

### Why Browser Automation Was Considered

- Browser can execute JavaScript
- Can wait for dynamic content to load
- Can interact with fully rendered DOM

### Why Browser Automation Is NOT Needed

- Found the internal AJAX API endpoint
- Can access content directly via HTTP GET
- Faster and simpler than browser automation
- No JavaScript execution required

---

## Solution Implementation

### Method: Direct AJAX API Access

**Advantages:**
- ✅ Fast (no browser overhead)
- ✅ Simple (standard HTTP request)
- ✅ Direct (exact content needed)
- ✅ No dependencies (just requests + BeautifulSoup)
- ✅ No JavaScript execution needed

**Implementation:**

```python
import requests
from bs4 import BeautifulSoup

def extract_talk(hash_fragment):
    # Parse hash: #:t210c:g8be
    hash_value = hash_fragment.replace('#:', '')
    encoded_talk = hash_value[1:]  # Remove 't' prefix

    # Handle query params and anchors
    if '&' in encoded_talk:
        encoded_talk = encoded_talk.split('&')[0]
    if '$' in encoded_talk:
        encoded_talk = encoded_talk.split('$')[0]

    # Extract conference ID
    conf_hex = encoded_talk.split(':')[0]
    talk_id = int(conf_hex, 16)

    # Fetch from API
    url = f"https://scriptures.byu.edu/content/talks_ajax/{talk_id}"
    response = requests.get(url)

    # Parse HTML
    soup = BeautifulSoup(response.text, 'html.parser')

    return {
        "title": soup.select_one('h1[data-aid]').get_text(strip=True),
        "speaker": soup.select_one('.author-name').get_text(strip=True),
        "calling": soup.select_one('.author-role').get_text(strip=True),
        "kicker": soup.select_one('.kicker').get_text(strip=True),
        "paragraphs": [p.get_text(strip=True)
                      for p in soup.select('.body-block p[data-aid]')]
    }
```

---

## Content Structure

### HTML Returned by API

```html
<div id="centernavbar">
    <div id="talklabel">2020–O:6, Russell M. Nelson, Moving Forward</div>
</div>

<div id="centercontent">
    <div id="talkcontent">
        <header>
            <h1 data-aid="144615376">Moving Forward</h1>
            <p class="author-name">By President Russell M. Nelson</p>
            <p class="author-role">President of The Church of Jesus Christ...</p>
            <p class="kicker">The work of the Lord is steadily moving forward.</p>
        </header>

        <div class="body-block">
            <p data-aid="144615380">My dear brothers and sisters...</p>
            <p data-aid="144615381">How grateful we are...</p>
            <!-- More paragraphs -->
        </div>

        <footer class="notes">
            <ol class="decimal">
                <li>Scripture references...</li>
            </ol>
        </footer>
    </div>
</div>
```

### Key Selectors

- **Title:** `h1[data-aid]`
- **Speaker:** `.author-name`
- **Calling:** `.author-role`
- **Kicker:** `.kicker`
- **Paragraphs:** `.body-block p[data-aid]`
- **Scripture References:** `.citation a`
- **Conference Info:** `#talklabel`

---

## Files Created

| File | Purpose | Location |
|------|---------|----------|
| `debug_extraction.py` | Initial debug script | `/Users/justinjones/Developer/temple-square/tmp-agents/debug_extraction.py` |
| `debug-raw-html.html` | Initial page HTML shell | `/Users/justinjones/Developer/temple-square/tmp-agents/debug-raw-html.html` |
| `talk-content.html` | API response HTML | `/Users/justinjones/Developer/temple-square/tmp-agents/talk-content.html` |
| `debug-extraction.json` | Final extracted data | `/Users/justinjones/Developer/temple-square/tmp-agents/debug-extraction.json` |
| `DEBUG-SUMMARY.md` | This summary | `/Users/justinjones/Developer/temple-square/tmp-agents/DEBUG-SUMMARY.md` |

---

## Recommendations

### For Production Implementation

1. **Use AJAX API Endpoint Directly**
   - Faster than browser automation
   - Simpler implementation
   - No Chrome DevTools MCP needed

2. **Hash Decoding Pattern**
   ```
   #:t{conference_hex}:{talk_additional_data}
   → /content/talks_ajax/{int(conference_hex, 16)}
   ```

3. **Content Parsing**
   - Use BeautifulSoup with the selectors documented above
   - Preserve `data-aid` attributes for reference linking
   - Extract scripture citations from `.citation` elements

4. **Edge Cases to Handle**
   - Query parameters (after `&` in hash)
   - Target anchors (after `$` in hash)
   - "About" page (special case)
   - October conferences (may have different hex encoding)

---

## Conclusion

**Problem:** JavaScript SPA with dynamic content loading prevented static extraction

**Solution:** Discovered internal AJAX API endpoint by analyzing JavaScript source

**Result:** ✅ Successful extraction without browser automation

**Performance:** Direct API access is faster and simpler than browser automation

**Chrome DevTools MCP Status:** Not required for this use case

---

## Example Output

**Talk Successfully Extracted:**

- **Conference:** October 2020, Session 6
- **Speaker:** President Russell M. Nelson
- **Title:** Moving Forward
- **Paragraphs:** 12
- **Scripture References:** 3 unique (Doctrine and Covenants 84:88, Helaman 3:29, 2 Nephi 1:15)

**First Paragraph:**
> My dear brothers and sisters, what a joy it is to be with you as we begin the 190th Semiannual General Conference of The Church of Jesus Christ of Latter-day Saints. I love joining with you in your homes or wherever you are to listen together to the messages of prophets, seers, and revelators and other Church leaders.

---

## Tools Used in Investigation

**Did NOT Use:**
- ❌ Chrome DevTools MCP (not available)
- ❌ WebFetch (would have failed - no JS execution)

**Did Use:**
- ✅ Bash - Command execution
- ✅ Python requests - HTTP requests
- ✅ BeautifulSoup - HTML parsing
- ✅ Read/Write - File operations
- ✅ Manual JavaScript analysis - Reverse engineering

---

**Investigation Complete**
All findings documented in: `/Users/justinjones/Developer/temple-square/tmp-agents/`
