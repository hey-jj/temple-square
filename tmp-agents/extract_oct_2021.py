#!/usr/bin/env python3
"""
Extract all talks from October 2021 General Conference using AJAX API
"""

import json
import time
import requests
from bs4 import BeautifulSoup
from pathlib import Path

# API endpoint
AJAX_API = "https://scriptures.byu.edu/content/talks_ajax/{decimal_id}"

# Output directory
OUTPUT_DIR = Path("/Users/justinjones/Developer/temple-square/tmp/data/talks/2021-O")

# All talk hex IDs from October 2021 (2151 through 2177)
TALK_IDS = [
    "2151", "2152", "2153", "2154", "2155", "2156", "2157", "2158", "2159",
    "215a", "215b", "215c", "215d", "215e", "215f", "2160", "2161", "2162",
    "2163", "2164", "2165", "2166", "2167", "2168", "2169", "216a", "216b",
    "216c", "216d", "216e", "216f", "2170", "2171", "2172", "2173", "2174",
    "2175", "2176", "2177"
]

# Expected talk information
TALK_INFO = {
    "2151": "Russell M. Nelson - Pure Truth, Pure Doctrine, and Pure Revelation",
    "2152": "Jeffrey R. Holland - The Greatest Possession",
    "2153": "Bonnie H. Cordon - Come unto Christ and Don't Come Alone",
    "2154": "Ulisses Soares - The Savior's Abiding Compassion",
    "2155": "D. Todd Christofferson - The Love of God",
    "2156": "Clark G. Gilbert - Becoming More in Christ: The Parable of the Slope",
    "2157": "Patricio M. Giuffra - A Faithful Search Rewarded",
    "2158": "Dallin H. Oaks - The Need for a Church",
    "2159": "David A. Bednar - With the Power of God in Great Glory",
    "215a": "Ciro Schmeil - Faith to Act and Become",
    "215b": "Susan H. Porter - God's Love: The Most Joyous to the Soul",
    "215c": "Erich W. Kopischke - Addressing Mental Health",
    "215d": "Ronald A. Rasband - The Things of My Soul",
    "215e": "Christoffel Golden, Jr. - Preparing for the Second Coming",
    "215f": "Moises Villanueva - Favored of the Lord in All My Days",
    "2160": "Gary E. Stevenson - Simply Beautiful—Beautifully Simple",
    "2161": "M. Russell Ballard - \"Lovest Thou Me More Than These?\"",
    "2162": "Sharon Eubank - I Pray He'll Use Us",
    "2163": "Brent H. Nielson - Is There No Balm in Gilead?",
    "2164": "Arnulfo Valenzuela - Deepening Our Conversion",
    "2165": "Bradley R. Wilcox - Worthiness Is Not Flawlessness",
    "2166": "Alfred Kyungu - To Be a Follower of Christ",
    "2167": "Marcus B. Nash - Hold Up Your Light",
    "2168": "Henry B. Eyring - The Faith to Ask and Then to Act",
    "2169": "Dieter F. Uchtdorf - Daily Restoration",
    "216a": "Camille N. Johnson - Invite Christ to Author Your Story",
    "216b": "Dale G. Renlund - The Peace of Christ Abolishes Enmity",
    "216c": "Vaiangina Sikahema - A House of Sequential Order",
    "216d": "Quentin L. Cook - Personal Peace in Challenging Times",
    "216e": "Russell M. Nelson - The Temple and Your Spiritual Foundation",
    "216f": "Gerrit W. Gong - Trust Again",
    "2170": "L. Todd Budge - Giving Holiness to the Lord",
    "2171": "Anthony D. Perkins - Remember Thy Suffering Saints",
    "2172": "Michael A. Dunn - One Percent Better",
    "2173": "Sean Douglas - Facing Our Spiritual Hurricanes",
    "2174": "Carlos G. Revillo, Jr. - Miracles of the Gospel",
    "2175": "Alvin F. Meredith III - Look down the Road",
    "2176": "Neil L. Andersen - The Name of the Church Is Not Negotiable",
    "2177": "Russell M. Nelson - Make Time for the Lord"
}


def extract_talk_content(soup):
    """Extract talk content from BeautifulSoup object."""
    paragraphs = []

    # Find all paragraph elements
    for p in soup.find_all('p'):
        text = p.get_text(strip=True)
        if text:
            paragraphs.append(text)

    return "\n\n".join(paragraphs)


def extract_metadata(soup, hex_id):
    """Extract metadata from the talk HTML."""
    metadata = {
        "talk_id": hex_id,
        "title": "",
        "speaker": "",
        "calling": "",
        "session": "",
        "conference": "October 2021 General Conference",
        "date": "October 2021"
    }

    # Try to extract title
    title_elem = soup.find('h1') or soup.find('h2', class_='title')
    if title_elem:
        metadata["title"] = title_elem.get_text(strip=True)

    # Try to extract speaker and calling
    speaker_elem = soup.find('p', class_='author-name') or soup.find('div', class_='author')
    if speaker_elem:
        metadata["speaker"] = speaker_elem.get_text(strip=True)

    calling_elem = soup.find('p', class_='author-role') or soup.find('div', class_='calling')
    if calling_elem:
        metadata["calling"] = calling_elem.get_text(strip=True)

    return metadata


def fetch_talk(hex_id):
    """Fetch a single talk using the AJAX API."""
    # Convert hex to decimal
    decimal_id = int(hex_id, 16)

    # Build API URL
    url = AJAX_API.format(decimal_id=decimal_id)

    print(f"Fetching talk {hex_id} (decimal: {decimal_id})...")
    print(f"  Expected: {TALK_INFO.get(hex_id, 'Unknown')}")
    print(f"  URL: {url}")

    try:
        # Fetch from API
        response = requests.get(url, timeout=30)
        response.raise_for_status()

        # Parse HTML
        soup = BeautifulSoup(response.text, 'html.parser')

        # Extract metadata and content
        metadata = extract_metadata(soup, hex_id)
        content = extract_talk_content(soup)

        # Build final structure
        talk_data = {
            **metadata,
            "content": content,
            "word_count": len(content.split()),
            "paragraph_count": len(content.split("\n\n")),
            "source_url": f"https://scriptures.byu.edu/content/talks/{hex_id}",
            "ajax_url": url,
            "extraction_method": "AJAX API"
        }

        # Save to file
        output_file = OUTPUT_DIR / f"{hex_id}.json"
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(talk_data, f, indent=2, ensure_ascii=False)

        print(f"  ✓ Saved: {output_file}")
        print(f"  Title: {metadata['title']}")
        print(f"  Speaker: {metadata['speaker']}")
        print(f"  Words: {talk_data['word_count']}, Paragraphs: {talk_data['paragraph_count']}")

        return True, talk_data

    except Exception as e:
        print(f"  ✗ Error: {e}")
        return False, None


def main():
    """Main extraction process."""
    print("=" * 80)
    print("Extracting October 2021 General Conference Talks")
    print("=" * 80)
    print(f"Total talks to extract: {len(TALK_IDS)}")
    print(f"Output directory: {OUTPUT_DIR}")
    print()

    successful = 0
    failed = 0
    failed_ids = []

    for idx, hex_id in enumerate(TALK_IDS, 1):
        print(f"\n[{idx}/{len(TALK_IDS)}] Processing talk {hex_id}...")

        success, talk_data = fetch_talk(hex_id)

        if success:
            successful += 1
        else:
            failed += 1
            failed_ids.append(hex_id)

        # Add delay between requests (except after last one)
        if idx < len(TALK_IDS):
            time.sleep(0.5)

    # Final summary
    print("\n" + "=" * 80)
    print("EXTRACTION COMPLETE")
    print("=" * 80)
    print(f"Successful: {successful}/{len(TALK_IDS)}")
    print(f"Failed: {failed}/{len(TALK_IDS)}")

    if failed_ids:
        print(f"Failed IDs: {', '.join(failed_ids)}")

    print()
    print(f"All extracted talks saved to: {OUTPUT_DIR}")


if __name__ == "__main__":
    main()
