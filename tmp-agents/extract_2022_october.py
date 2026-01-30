#!/usr/bin/env python3
"""
Extract all 35 talks from October 2022 General Conference using AJAX API
API Endpoint: https://scriptures.byu.edu/content/talks_ajax/{decimal_id}
"""

import requests
import json
import time
from bs4 import BeautifulSoup
from pathlib import Path

# Conference metadata
CONFERENCE_YEAR = 2022
CONFERENCE_MONTH = "October"
CONFERENCE_CODE = "2022-O"

# Output directory
OUTPUT_DIR = Path("/Users/justinjones/Developer/temple-square/tmp/data/talks/2022-O")
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

# All talks with hex IDs and metadata
TALKS = [
    {"hex_id": "219b", "title": "Helping the Poor and Distressed", "speaker": "Dallin H. Oaks"},
    {"hex_id": "219c", "title": "Jesus Christ Is the Strength of Youth", "speaker": "Dieter F. Uchtdorf"},
    {"hex_id": "219d", "title": "Seeing More of Jesus Christ in Our Lives", "speaker": "Tracy Y. Browning"},
    {"hex_id": "219e", "title": "A Framework for Personal Revelation", "speaker": "Dale G. Renlund"},
    {"hex_id": "219f", "title": "Let Doing Good Be Our Normal", "speaker": "Rafael E. Pino"},
    {"hex_id": "21a0", "title": "The Eternal Principle of Love", "speaker": "Hugo Montoya"},
    {"hex_id": "21a1", "title": "This Day", "speaker": "Ronald A. Rasband"},
    {"hex_id": "21a2", "title": "What Is True?", "speaker": "Russell M. Nelson"},
    {"hex_id": "21a3", "title": "Follow Jesus Christ with Footsteps of Faith", "speaker": "M. Russell Ballard"},
    {"hex_id": "21a4", "title": "Beauty for Ashes", "speaker": "Kristin M. Yee"},
    {"hex_id": "21a5", "title": "Be Perfected in Him", "speaker": "Paul V. Johnson"},
    {"hex_id": "21a6", "title": "In Partnership with the Lord", "speaker": "Ulisses Soares"},
    {"hex_id": "21a7", "title": "And They Sought to See Jesus", "speaker": "James W. McConkie III"},
    {"hex_id": "21a8", "title": "Building a Life Resistant to the Adversary", "speaker": "Jorge F. Zeballos"},
    {"hex_id": "21a9", "title": "The Doctrine of Belonging", "speaker": "D. Todd Christofferson"},
    {"hex_id": "21aa", "title": "Our Earthly Stewardship", "speaker": "Gérald Caussé"},
    {"hex_id": "21ab", "title": "Wholehearted", "speaker": "Michelle D. Craig"},
    {"hex_id": "21ac", "title": "Are You Still Willing?", "speaker": "Kevin W. Pearson"},
    {"hex_id": "21ad", "title": "Courage to Proclaim the Truth", "speaker": "Denelson Silva"},
    {"hex_id": "21ae", "title": "Drawing Closer to the Savior", "speaker": "Neil L. Andersen"},
    {"hex_id": "21af", "title": "Lifted Up upon the Cross", "speaker": "Jeffrey R. Holland"},
    {"hex_id": "21b0", "title": "His Yoke Is Easy", "speaker": "J. Anette Dennis"},
    {"hex_id": "21b1", "title": "Happy and Forever", "speaker": "Gerrit W. Gong"},
    {"hex_id": "21b2", "title": "Patterns of Discipleship", "speaker": "Joseph W. Sitati"},
    {"hex_id": "21b3", "title": "Lasting Discipleship", "speaker": "Steven J. Lund"},
    {"hex_id": "21b4", "title": "Put On Thy Strength, O Zion", "speaker": "David A. Bednar"},
    {"hex_id": "21b5", "title": "Overcome the World and Find Rest", "speaker": "Russell M. Nelson"},
    {"hex_id": "21b6", "title": "Legacy of Encouragement", "speaker": "Henry B. Eyring"},
    {"hex_id": "21b7", "title": "The Answer Is Jesus", "speaker": "Ryan K. Olsen"},
    {"hex_id": "21b8", "title": "That They Might Know Thee", "speaker": "Jonathan S. Schmitt"},
    {"hex_id": "21b9", "title": "The Virtue of the Word", "speaker": "Mark D. Eddy"},
    {"hex_id": "21ba", "title": "Nourishing and Bearing Your Testimony", "speaker": "Gary E. Stevenson"},
    {"hex_id": "21bb", "title": "We Can Do Hard Things through Him", "speaker": "Isaac K. Morrison"},
    {"hex_id": "21bc", "title": "Be True to God and His Work", "speaker": "Quentin L. Cook"},
    {"hex_id": "21bd", "title": "Focus on the Temple", "speaker": "Russell M. Nelson"},
]


def extract_text_content(html_content):
    """Extract clean text content from HTML, preserving paragraph structure"""
    soup = BeautifulSoup(html_content, 'html.parser')

    # Remove script and style elements
    for script in soup(["script", "style"]):
        script.decompose()

    # Get text and preserve some structure
    text = soup.get_text(separator='\n\n', strip=True)

    # Clean up multiple newlines
    lines = [line.strip() for line in text.split('\n') if line.strip()]
    cleaned_text = '\n\n'.join(lines)

    return cleaned_text


def fetch_talk_from_ajax(hex_id, decimal_id):
    """Fetch talk content from AJAX API"""
    url = f"https://scriptures.byu.edu/content/talks_ajax/{decimal_id}"

    headers = {
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
        'Accept': 'application/json, text/html, */*',
        'X-Requested-With': 'XMLHttpRequest'
    }

    try:
        response = requests.get(url, headers=headers, timeout=30)
        response.raise_for_status()

        # The API returns HTML content
        html_content = response.text

        # Parse and extract clean text
        content_text = extract_text_content(html_content)

        return content_text, html_content

    except Exception as e:
        print(f"  ERROR fetching talk: {e}")
        return None, None


def extract_talk(talk_info):
    """Extract a single talk and save to JSON"""
    hex_id = talk_info["hex_id"]
    title = talk_info["title"]
    speaker = talk_info["speaker"]

    # Convert hex to decimal
    decimal_id = int(hex_id, 16)

    print(f"Extracting {hex_id} ({decimal_id}): {speaker} - {title}")

    # Fetch from AJAX API
    content, raw_html = fetch_talk_from_ajax(hex_id, decimal_id)

    if content is None:
        print(f"  FAILED to extract talk {hex_id}")
        return False

    # Build output structure matching canonical schema
    output = {
        "talk_id": hex_id,
        "decimal_id": decimal_id,
        "speaker": speaker,
        "title": title,
        "conference": f"{CONFERENCE_MONTH} {CONFERENCE_YEAR}",
        "content": content,
        "source_url": f"https://scriptures.byu.edu/content/talks_ajax/{decimal_id}",
        "extraction_method": "AJAX API"
    }

    # Save to JSON file
    output_path = OUTPUT_DIR / f"{hex_id}.json"
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(output, f, indent=2, ensure_ascii=False)

    print(f"  ✓ Saved to {output_path}")
    print(f"  Content length: {len(content)} characters")

    return True


def main():
    """Extract all talks from October 2022 General Conference"""
    print("="*80)
    print(f"Extracting {len(TALKS)} talks from {CONFERENCE_MONTH} {CONFERENCE_YEAR} General Conference")
    print(f"Output directory: {OUTPUT_DIR}")
    print("="*80)
    print()

    successful = 0
    failed = 0

    for i, talk_info in enumerate(TALKS, 1):
        print(f"[{i}/{len(TALKS)}] ", end="")

        success = extract_talk(talk_info)

        if success:
            successful += 1
        else:
            failed += 1

        # Add delay between requests (0.5 seconds)
        if i < len(TALKS):
            time.sleep(0.5)

        print()

    # Summary
    print("="*80)
    print("EXTRACTION COMPLETE")
    print(f"Successful: {successful}/{len(TALKS)}")
    print(f"Failed: {failed}/{len(TALKS)}")
    print(f"Output directory: {OUTPUT_DIR}")
    print("="*80)

    # List saved files
    saved_files = sorted(OUTPUT_DIR.glob("*.json"))
    print(f"\nSaved {len(saved_files)} files:")
    for file in saved_files:
        print(f"  - {file.name}")


if __name__ == "__main__":
    main()
