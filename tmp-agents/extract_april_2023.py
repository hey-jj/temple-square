#!/usr/bin/env python3
"""
Extract April 2023 General Conference talks using AJAX API
"""
import json
import time
import requests
from bs4 import BeautifulSoup
from pathlib import Path

# Output directory
OUTPUT_DIR = Path("/Users/justinjones/Developer/temple-square/tmp/data/talks/2023-A")
OUTPUT_DIR.mkdir(parents=True, exist_ok=True)

# Talk definitions (hex_id: speaker - title)
TALKS = {
    "21be": "Gary E. Stevenson - The Greatest Easter Story Ever Told",
    "21bf": "Bonnie H. Cordon - Never Give Up an Opportunity to Testify",
    "21c0": "Carl B. Cook - Just Keep Going—with Faith",
    "21c1": "Gerrit W. Gong - Ministering",
    "21c2": "Quentin L. Cook - Safely Gathered Home",
    "21c3": "Allen D. Haynie - A Living Prophet for the Latter Days",
    "21c4": "Henry B. Eyring - Finding Personal Peace",
    "21c5": "Dale G. Renlund - Accessing God's Power through Covenants",
    "21c6": "Peter F. Meurs - He Could Heal Me!",
    "21c7": "Randall K. Bennett - Your Patriarchal Blessing",
    "21c8": "Craig C. Christensen - There Can Be Nothing So Exquisite",
    "21c9": "Evan A. Schmutz - Trusting the Doctrine of Christ",
    "21ca": "Benjamín De Hoyos - The Work of the Temple and Family History",
    "21cb": "Dieter F. Uchtdorf - Jesus Christ Is the Strength of Parents",
    "21cc": "Mark A. Bragg - Christlike Poise",
    "21cd": "Milton Camargo - Focus on Jesus Christ",
    "21ce": "K. Brett Nattress - Have I Truly Been Forgiven?",
    "21cf": "Juan A. Uceda - The Lord Jesus Christ Teaches Us to Minister",
    "21d0": "D. Todd Christofferson - One in Christ",
    "21d1": "Camille N. Johnson - Jesus Christ Is Relief",
    "21d2": "Ulisses Soares - Followers of the Prince of Peace",
    "21d3": "Kazuhiko Yamashita - When to Receive Your Patriarchal Blessing",
    "21d4": "Neil L. Andersen - My Mind Caught Hold upon This Thought",
    "21d5": "Kevin R. Duncan - A Voice of Gladness!",
    "21d6": "Russell M. Nelson - Peacemakers Needed",
    "21d7": "Dallin H. Oaks - The Teachings of Jesus Christ",
    "21d8": "M. Russell Ballard - Remember What Matters Most",
    "21d9": "Ronald A. Rasband - Hosanna to the Most High God",
    "21da": "Vern P. Stanfill - The Imperfect Harvest",
    "21db": "W. Mark Bassett - After the Fourth Day",
    "21dc": "Ahmad S. Corbitt - Do You Know Why I as a Christian Believe in Christ?",
    "21dd": "David A. Bednar - Abide in Me, and I in You",
    "21de": "Russell M. Nelson - The Answer Is Always Jesus Christ",
}

def fetch_talk_ajax(hex_id):
    """Fetch talk content from AJAX API"""
    decimal_id = int(hex_id, 16)
    url = f"https://scriptures.byu.edu/content/talks_ajax/{decimal_id}"

    print(f"Fetching {hex_id} (decimal: {decimal_id})...")
    response = requests.get(url, timeout=30)
    response.raise_for_status()
    return response.text

def parse_talk_html(html_content, hex_id):
    """Parse talk HTML and extract structured data"""
    soup = BeautifulSoup(html_content, 'html.parser')

    # Extract title
    title_elem = soup.find('h1')
    title = title_elem.get_text(strip=True) if title_elem else ""

    # Extract speaker and calling
    byline_elem = soup.find('p', class_='author-name')
    speaker = ""
    calling = ""
    if byline_elem:
        speaker = byline_elem.get_text(strip=True)
        role_elem = soup.find('p', class_='author-role')
        if role_elem:
            calling = role_elem.get_text(strip=True)

    # Extract paragraphs (full text)
    paragraphs = []
    content_div = soup.find('div', class_='body-block')
    if content_div:
        for p in content_div.find_all('p'):
            text = p.get_text(strip=True)
            if text:
                paragraphs.append(text)

    # Extract session info
    session_elem = soup.find('p', class_='kicker')
    session = session_elem.get_text(strip=True) if session_elem else ""

    # Build canonical schema
    talk_data = {
        "id": hex_id,
        "title": title,
        "speaker": speaker,
        "calling": calling,
        "session": session,
        "conference": "April 2023 General Conference",
        "date": "2023-04",
        "text": "\n\n".join(paragraphs),
        "paragraphs": paragraphs,
        "source_url": f"https://www.churchofjesuschrist.org/study/general-conference/2023/04/{hex_id}",
        "ajax_url": f"https://scriptures.byu.edu/content/talks_ajax/{int(hex_id, 16)}"
    }

    return talk_data

def main():
    """Extract all talks"""
    success_count = 0
    failed = []

    print(f"Extracting {len(TALKS)} talks from April 2023 General Conference\n")

    for i, (hex_id, description) in enumerate(TALKS.items(), 1):
        try:
            print(f"[{i}/{len(TALKS)}] {hex_id}: {description}")

            # Fetch from AJAX API
            html_content = fetch_talk_ajax(hex_id)

            # Parse and structure data
            talk_data = parse_talk_html(html_content, hex_id)

            # Save to JSON
            output_file = OUTPUT_DIR / f"{hex_id}.json"
            with open(output_file, 'w', encoding='utf-8') as f:
                json.dump(talk_data, f, indent=2, ensure_ascii=False)

            print(f"  ✓ Saved to {output_file.name}")
            print(f"  Speaker: {talk_data['speaker']}")
            print(f"  Paragraphs: {len(talk_data['paragraphs'])}")
            success_count += 1

            # Delay between requests
            if i < len(TALKS):
                time.sleep(0.5)

        except Exception as e:
            print(f"  ✗ ERROR: {e}")
            failed.append((hex_id, description, str(e)))

        print()

    # Summary
    print("=" * 70)
    print(f"EXTRACTION COMPLETE")
    print(f"Successfully extracted: {success_count}/{len(TALKS)} talks")

    if failed:
        print(f"\nFailed extractions ({len(failed)}):")
        for hex_id, desc, error in failed:
            print(f"  - {hex_id}: {desc}")
            print(f"    Error: {error}")
    else:
        print("\nAll talks extracted successfully!")

    print(f"\nOutput directory: {OUTPUT_DIR}")

if __name__ == "__main__":
    main()
