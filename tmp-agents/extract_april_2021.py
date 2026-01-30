#!/usr/bin/env python3
"""
Extract all talks from April 2021 General Conference using AJAX API
"""

import json
import requests
import time
from bs4 import BeautifulSoup
from pathlib import Path

# Talk metadata
TALKS = [
    ("212e", "Russell M. Nelson", "Welcome Message"),
    ("212f", "Dieter F. Uchtdorf", "God among Us"),
    ("2130", "Joy D. Jones", "Essential Conversations"),
    ("2131", "Jan E. Newman", "Teaching in the Savior's Way"),
    ("2132", "Gary E. Stevenson", "Hearts Knit Together"),
    ("2133", "Gerrit W. Gong", "Room in the Inn"),
    ("2134", "Henry B. Eyring", "I Love to See the Temple"),
    ("2135", "Jeffrey R. Holland", "Not as the World Giveth"),
    ("2136", "Jorge T. Becerra", "Poor Little Ones"),
    ("2137", "Dale G. Renlund", "Infuriating Unfairness"),
    ("2138", "Neil L. Andersen", "The Personal Journey of a Child of God"),
    ("2139", "Thierry K. Mutombo", "Ye Shall Be Free"),
    ("213a", "M. Russell Ballard", "Hope in Christ"),
    ("213b", "Quentin L. Cook", "Bishops—Shepherds over the Lord's Flock"),
    ("213c", "Ahmad S. Corbitt", "You Can Gather Israel!"),
    ("213d", "S. Gifford Nielsen", "This Is Our Time!"),
    ("213e", "Henry B. Eyring", "Bless in His Name"),
    ("213f", "Dallin H. Oaks", "What Has Our Savior Done for Us?"),
    ("2140", "Russell M. Nelson", "What We Are Learning and Will Never Forget"),
    ("2141", "Ulisses Soares", "Jesus Christ: The Caregiver of Our Soul"),
    ("2142", "Reyna I. Aburto", "The Grave Has No Victory"),
    ("2143", "S. Mark Palmer", "Our Sorrow Shall Be Turned into Joy"),
    ("2144", "Edward Dube", "Pressing toward the Mark"),
    ("2145", "Jose A. Teixeira", "Remember Your Way Back Home"),
    ("2146", "Taniela B. Wakolo", "God Loves His Children"),
    ("2147", "Chi Hong (Sam) Wong", "They Cannot Prevail; We Cannot Fall"),
    ("2148", "Michael J. Teh", "Our Personal Savior"),
    ("2149", "Russell M. Nelson", "Christ Is Risen; Faith in Him Will Move Mountains"),
    ("214a", "Dallin H. Oaks", "Defending Our Divinely Inspired Constitution"),
    ("214b", "Ronald A. Rasband", "\"Behold! I Am a God of Miracles\""),
    ("214c", "Timothy J. Dyches", "Light Cleaveth unto Light"),
    ("214d", "D. Todd Christofferson", "Why the Covenant Path"),
    ("214e", "Alan R. Walker", "The Gospel Light of Truth and Love"),
    ("214f", "David A. Bednar", "\"The Principles of My Gospel\""),
    ("2150", "Russell M. Nelson", "COVID-19 and Temples"),
]

OUTPUT_DIR = Path("/Users/justinjones/Developer/temple-square/tmp/data/talks/2021-A")
AJAX_API_URL = "https://scriptures.byu.edu/content/talks_ajax/{}"

def extract_talk_content(html_content):
    """Parse HTML and extract talk content"""
    soup = BeautifulSoup(html_content, 'html.parser')

    # Extract paragraphs
    paragraphs = []
    for p in soup.find_all('p'):
        text = p.get_text(strip=True)
        if text:
            paragraphs.append(text)

    # Join all paragraphs
    full_text = "\n\n".join(paragraphs)

    return full_text

def fetch_talk(hex_id, speaker, title):
    """Fetch a single talk from the AJAX API"""
    # Convert hex to decimal
    decimal_id = int(hex_id, 16)

    # Build API URL
    url = AJAX_API_URL.format(decimal_id)

    print(f"Fetching {hex_id} ({decimal_id}): {speaker} - {title}")

    try:
        response = requests.get(url, timeout=30)
        response.raise_for_status()

        # Parse the HTML content
        html_content = response.text
        talk_text = extract_talk_content(html_content)

        # Build the data structure
        talk_data = {
            "talk_id": hex_id,
            "decimal_id": decimal_id,
            "speaker": speaker,
            "title": title,
            "conference": "April 2021",
            "content": talk_text,
            "source_url": url,
            "extraction_method": "AJAX API"
        }

        # Save to JSON file
        output_file = OUTPUT_DIR / f"{hex_id}.json"
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(talk_data, f, indent=2, ensure_ascii=False)

        print(f"  ✓ Saved to {output_file}")
        return True

    except Exception as e:
        print(f"  ✗ Error fetching {hex_id}: {e}")
        return False

def main():
    """Extract all talks"""
    print(f"Starting extraction of {len(TALKS)} talks from April 2021 General Conference")
    print(f"Output directory: {OUTPUT_DIR}")
    print("-" * 80)

    success_count = 0
    fail_count = 0

    for hex_id, speaker, title in TALKS:
        if fetch_talk(hex_id, speaker, title):
            success_count += 1
        else:
            fail_count += 1

        # Delay between requests
        time.sleep(0.5)

    print("-" * 80)
    print(f"Extraction complete!")
    print(f"  Success: {success_count}/{len(TALKS)}")
    print(f"  Failed: {fail_count}/{len(TALKS)}")

    if success_count == len(TALKS):
        print("\n✓ All talks extracted successfully!")
    else:
        print(f"\n⚠ {fail_count} talk(s) failed to extract")

if __name__ == "__main__":
    main()
