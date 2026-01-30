#!/usr/bin/env python3
"""
Extract all talks from April 2022 General Conference using AJAX API.
"""

import json
import time
import re
from urllib.request import urlopen, Request
from bs4 import BeautifulSoup
from datetime import datetime

# Talk metadata
TALKS = [
    ("2178", "Russell M. Nelson", "Preaching the Gospel of Peace"),
    ("2179", "M. Russell Ballard", "Missionary Service Blessed My Life Forever"),
    ("217a", "Reyna I. Aburto", "We Are The Church of Jesus Christ"),
    ("217b", "David A. Bednar", "But We Heeded Them Not"),
    ("217c", "Neil L. Andersen", "Following Jesus: Being a Peacemaker"),
    ("217d", "Eduardo Gavarret", "A Mighty Change of Heart"),
    ("217e", "Larry S. Kacher", "Ladder of Faith"),
    ("217f", "Henry B. Eyring", "Steady in the Storms"),
    ("2180", "Jeffrey R. Holland", "Fear Not: Believe Only!"),
    ("2181", "Patrick Kearon", "He Is Risen with Healing in His Wings"),
    ("2182", "Marcos A. Aidukaitis", "Lift Up Your Heart and Rejoice"),
    ("2183", "Gerrit W. Gong", "We Each Have a Story"),
    ("2184", "Adrian Ochoa", "Is the Plan Working?"),
    ("2185", "Kevin S. Hamilton", "Then Will I Make Weak Things Become Strong"),
    ("2186", "Quentin L. Cook", "Conversion to the Will of God"),
    ("2187", "Dallin H. Oaks", "Introductory Message"),
    ("2188", "Susan H. Porter", "Lessons at the Well"),
    ("2189", "Rebecca L. Craven", "Do What Mattereth Most"),
    ("218a", "Jean B. Bingham", "Covenants with God Strengthen, Protect, and Prepare Us"),
    ("218b", "Dale G. Renlund", "Your Divine Nature and Eternal Destiny"),
    ("218c", "D. Todd Christofferson", "Our Relationship with God"),
    ("218d", "Amy A. Wright", "Christ Heals That Which Is Broken"),
    ("218e", "Gary E. Stevenson", "Love, Share, Invite"),
    ("218f", "Michael T. Ringwood", "For God So Loved Us"),
    ("2190", "Ronald A. Rasband", "To Heal the World"),
    ("2191", "Hugo E. Martinez", "Teaching Self-Reliance to Children and Youth"),
    ("2192", "Russell M. Nelson", "The Power of Spiritual Momentum"),
    ("2193", "Dallin H. Oaks", "Divine Love in the Father's Plan"),
    ("2194", "Adeyinka A. Ojediran", "The Covenant Path: The Way to Eternal Life"),
    ("2195", "Jorg Klebingat", "Valiant Discipleship in the Latter Days"),
    ("2196", "Mark L. Pace", "Conversion Is Our Goal"),
    ("2197", "Ulisses Soares", "In Awe of Christ and His Gospel"),
    ("2198", "Randy D. Funk", "Come into the Fold of God"),
    ("2199", "Dieter F. Uchtdorf", "Our Heartfelt All"),
    ("219a", "Russell M. Nelson", "Now Is the Time"),
]

OUTPUT_DIR = "/Users/justinjones/Developer/temple-square/tmp/data/talks/2022-A"
AJAX_API_BASE = "https://scriptures.byu.edu/content/talks_ajax"

def extract_talk(hex_id, expected_speaker, expected_title):
    """Extract a single talk using the AJAX API."""

    # Convert hex to decimal
    decimal_id = int(hex_id, 16)
    url = f"{AJAX_API_BASE}/{decimal_id}"

    print(f"Fetching {hex_id} ({decimal_id}): {expected_speaker} - {expected_title}")

    try:
        # Fetch from AJAX API
        req = Request(url, headers={'User-Agent': 'Mozilla/5.0'})
        with urlopen(req, timeout=30) as response:
            html_content = response.read().decode('utf-8')

        # Parse HTML with BeautifulSoup
        soup = BeautifulSoup(html_content, 'html.parser')

        # Extract title
        title_elem = soup.find('h1')
        title = title_elem.get_text(strip=True) if title_elem else expected_title

        # Extract speaker and calling
        speaker = expected_speaker
        calling = ""

        # Look for speaker info (typically in a subtitle or author div)
        speaker_elem = soup.find('p', class_='author-name')
        if not speaker_elem:
            speaker_elem = soup.find('p', class_='subtitle')
        if not speaker_elem:
            # Try to find any paragraph that might contain speaker info
            first_p = soup.find('p')
            if first_p and len(first_p.get_text(strip=True)) < 100:
                speaker_elem = first_p

        if speaker_elem:
            speaker_text = speaker_elem.get_text(strip=True)
            # Extract calling if present (typically after "By")
            if 'By ' in speaker_text:
                parts = speaker_text.split('By ', 1)[1]
                if ',' in parts or ' of ' in parts.lower():
                    # Format: "Name, Calling" or "Name of Calling"
                    speaker = parts.split(',')[0].strip() if ',' in parts else parts.split(' of ')[0].strip()
                    calling = parts.replace(speaker, '').strip(', ')

        # Extract content paragraphs
        content_paragraphs = []

        # Find main content area
        content_div = soup.find('div', class_='body-block')
        if not content_div:
            content_div = soup.find('div', class_='article-content')
        if not content_div:
            content_div = soup

        # Extract all paragraphs
        for p in content_div.find_all('p'):
            text = p.get_text(strip=True)
            # Skip empty paragraphs and likely metadata
            if text and len(text) > 20 and not text.startswith('By '):
                content_paragraphs.append(text)

        # Build full content
        content = '\n\n'.join(content_paragraphs)

        # Create canonical schema
        talk_data = {
            "talk_id": hex_id,
            "conference": "2022-A",
            "title": title,
            "speaker": speaker,
            "calling": calling,
            "content": content,
            "metadata": {
                "decimal_id": decimal_id,
                "extraction_date": datetime.now().isoformat(),
                "source": "ajax_api"
            }
        }

        # Save to file
        output_path = f"{OUTPUT_DIR}/{hex_id}.json"
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(talk_data, f, indent=2, ensure_ascii=False)

        print(f"  ✓ Saved to {hex_id}.json ({len(content)} chars)")
        return True

    except Exception as e:
        print(f"  ✗ Error: {e}")
        return False

def main():
    """Extract all talks."""
    print(f"Extracting {len(TALKS)} talks from April 2022 General Conference")
    print(f"Output directory: {OUTPUT_DIR}\n")

    success_count = 0
    failed_talks = []

    for i, (hex_id, speaker, title) in enumerate(TALKS, 1):
        print(f"[{i}/{len(TALKS)}] ", end='')

        if extract_talk(hex_id, speaker, title):
            success_count += 1
        else:
            failed_talks.append((hex_id, speaker, title))

        # Delay between requests (except after last one)
        if i < len(TALKS):
            time.sleep(0.5)

    # Summary
    print("\n" + "="*70)
    print(f"Extraction complete: {success_count}/{len(TALKS)} talks successfully extracted")

    if failed_talks:
        print(f"\nFailed talks ({len(failed_talks)}):")
        for hex_id, speaker, title in failed_talks:
            print(f"  - {hex_id}: {speaker} - {title}")
    else:
        print("\nAll talks extracted successfully!")

    print("="*70)

if __name__ == "__main__":
    main()
