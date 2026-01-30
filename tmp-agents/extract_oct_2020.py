#!/usr/bin/env python3
"""
Extract all talks from October 2020 General Conference using AJAX API
"""

import json
import time
import requests
from bs4 import BeautifulSoup
from pathlib import Path

# Talk metadata with hex IDs
TALKS = [
    ("210c", "Russell M. Nelson", "Moving Forward"),
    ("210d", "David A. Bednar", "We Will Prove Them Herewith"),
    ("210e", "Scott D. Whiting", "Becoming like Him"),
    ("210f", "Michelle D. Craig", "Eyes to See"),
    ("2110", "Quentin L. Cook", "Hearts Knit in Righteousness and Unity"),
    ("2111", "Ronald A. Rasband", "Recommended to the Lord"),
    ("2112", "Dallin H. Oaks", "Love Your Enemies"),
    ("2113", "D. Todd Christofferson", "Sustainable Societies"),
    ("2114", "Steven J. Lund", "Finding Joy in Christ"),
    ("2115", "Gerrit W. Gong", "All Nations, Kindreds, and Tongues"),
    ("2116", "W. Christopher Waddell", "There Was Bread"),
    ("2117", "Matthew S. Holland", "The Exquisite Gift of the Son"),
    ("2118", "William K. Jackson", "The Culture of Christ"),
    ("2119", "Dieter F. Uchtdorf", "God Will Do Something Unimaginable"),
    ("211a", "Sharon Eubank", "By Union of Feeling We Obtain Power with God"),
    ("211b", "Rebecca L. Craven", "Keep the Change"),
    ("211c", "Cristina B. Franco", "The Healing Power of Jesus Christ"),
    ("211d", "Henry B. Eyring", "Sisters in Zion"),
    ("211e", "Dallin H. Oaks", "Be of Good Cheer"),
    ("211f", "Russell M. Nelson", "Embrace the Future with Faith"),
    ("2120", "M. Russell Ballard", "Watch Ye Therefore, and Pray Always"),
    ("2121", "Lisa L. Harkness", "Peace, Be Still"),
    ("2122", "Ulisses Soares", "Seek Christ in Every Thought"),
    ("2123", "Carlos A. Godoy", "I Believe in Angels"),
    ("2124", "Neil L. Andersen", "We Talk of Christ"),
    ("2125", "Russell M. Nelson", "Let God Prevail"),
    ("2126", "Henry B. Eyring", "Tested, Proved, and Polished"),
    ("2127", "Jeremy R. Jaggi", "Let Patience Have Her Perfect Work"),
    ("2128", "Gary E. Stevenson", "Highly Favored of the Lord"),
    ("2129", "Milton Camargo", "Ask, Seek, and Knock"),
    ("212a", "Dale G. Renlund", "Do Justly, Love Mercy, and Walk Humbly with God"),
    ("212b", "Kelly R. Johnson", "Enduring Power"),
    ("212c", "Jeffrey R. Holland", "Waiting on the Lord"),
    ("212d", "Russell M. Nelson", "A New Normal"),
]

OUTPUT_DIR = Path("/Users/justinjones/Developer/temple-square/tmp/data/talks/2020-O")
API_BASE_URL = "https://scriptures.byu.edu/content/talks_ajax/{}"

def extract_talk(hex_id, expected_speaker, expected_title):
    """Extract a single talk from the AJAX API"""

    # Convert hex ID to decimal
    decimal_id = int(hex_id, 16)

    print(f"\n[{hex_id}] Extracting: {expected_speaker} - {expected_title}")
    print(f"  Decimal ID: {decimal_id}")

    # Fetch from AJAX API
    url = API_BASE_URL.format(decimal_id)
    print(f"  Fetching: {url}")

    try:
        response = requests.get(url, timeout=30)
        response.raise_for_status()
        html_content = response.text

        # Parse HTML with BeautifulSoup
        soup = BeautifulSoup(html_content, 'html.parser')

        # Extract talk content
        talk_data = {
            "id": hex_id,
            "decimal_id": decimal_id,
            "speaker": expected_speaker,
            "title": expected_title,
            "conference": "October 2020 General Conference",
            "url": f"https://www.churchofjesuschrist.org/study/general-conference/2020/10/{hex_id}",
            "ajax_url": url,
        }

        # Extract actual title from HTML
        title_elem = soup.find('h1')
        if title_elem:
            talk_data["title_extracted"] = title_elem.get_text(strip=True)

        # Extract speaker from HTML
        speaker_elem = soup.find('p', class_='author-name')
        if speaker_elem:
            talk_data["speaker_extracted"] = speaker_elem.get_text(strip=True)

        # Extract calling/subtitle
        calling_elem = soup.find('p', class_='author-role')
        if calling_elem:
            talk_data["calling"] = calling_elem.get_text(strip=True)

        # Extract all paragraphs
        paragraphs = []
        for p in soup.find_all('p'):
            # Skip author info paragraphs
            if 'author-name' in p.get('class', []) or 'author-role' in p.get('class', []):
                continue
            text = p.get_text(strip=True)
            if text:
                paragraphs.append(text)

        talk_data["paragraphs"] = paragraphs
        talk_data["paragraph_count"] = len(paragraphs)

        # Extract full text
        body = soup.find('div', class_='body') or soup.find('body') or soup
        talk_data["full_text"] = body.get_text(separator='\n\n', strip=True)

        # Extract kicker (session info)
        kicker_elem = soup.find('p', class_='kicker')
        if kicker_elem:
            talk_data["session"] = kicker_elem.get_text(strip=True)

        # Save to JSON file
        output_path = OUTPUT_DIR / f"{hex_id}.json"
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(talk_data, f, indent=2, ensure_ascii=False)

        print(f"  ✓ Saved to: {output_path}")
        print(f"  ✓ Paragraphs extracted: {len(paragraphs)}")

        return True

    except Exception as e:
        print(f"  ✗ Error: {e}")
        return False

def main():
    """Extract all talks"""
    print(f"Extracting {len(TALKS)} talks from October 2020 General Conference")
    print(f"Output directory: {OUTPUT_DIR}")
    print("=" * 80)

    successful = 0
    failed = 0

    for hex_id, speaker, title in TALKS:
        if extract_talk(hex_id, speaker, title):
            successful += 1
        else:
            failed += 1

        # Rate limiting delay
        time.sleep(0.5)

    print("\n" + "=" * 80)
    print("EXTRACTION COMPLETE")
    print(f"Total talks: {len(TALKS)}")
    print(f"Successful: {successful}")
    print(f"Failed: {failed}")
    print("=" * 80)

if __name__ == "__main__":
    main()
