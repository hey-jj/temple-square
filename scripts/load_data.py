#!/usr/bin/env python3
"""Load scripture and talk JSON data into Cloud SQL PostgreSQL."""

import json
import os
import re
from pathlib import Path
import psycopg2
from psycopg2.extras import execute_values

# Configuration
DB_HOST = "35.199.189.20"
DB_NAME = "conference"
DB_USER = "postgres"
DB_PASS = open("/tmp/temple-square-db-pass.txt").read().strip()

DATA_DIR = Path("/Users/justinjones/Developer/temple-square/tmp/data")
HEADSHOTS_BASE_URL = "https://storage.googleapis.com/temple-square-assets/headshots"

# Headshot mapping (name_slug -> files exist)
HEADSHOT_SPEAKERS = {
    "aldred-kyungu", "amy-wright", "andrea-spannaus", "camille-johnson",
    "christopher-waddell", "dale-renlund", "dallin-oaks", "david-bednar",
    "dieterf-uchtdorf", "edward-dube", "emily-freeman", "gary-stevenson",
    "gerald-causse", "gerrit-gong", "henry-eyring", "hugo-martinez",
    "j-anette-dennis", "joaquin-costa", "kristin-lee", "neil-andersen",
    "patrick-kearon", "quentin-cook", "ronald-rasband", "russell-nelson",
    "susan-porter", "tamara-runia", "todd-christofferson", "tracy-browning",
    "ulisses-soares"
}


def slugify(name: str) -> str:
    """Convert speaker name to slug format."""
    name = name.lower()
    # Remove titles
    name = re.sub(r'^(elder|president|sister|bishop)\s+', '', name)
    # Remove middle initials (e.g., "H." in "Dallin H. Oaks")
    name = re.sub(r'\s+[a-z]\.\s+', ' ', name)
    # Remove punctuation
    name = re.sub(r'[^\w\s-]', '', name)
    # Replace spaces with hyphens
    name = re.sub(r'\s+', '-', name.strip())
    return name


def get_headshot_urls(name_slug: str) -> tuple:
    """Return (portrait_url, square_url) if headshots exist."""
    if name_slug in HEADSHOT_SPEAKERS:
        return (
            f"{HEADSHOTS_BASE_URL}/{name_slug}-portrait.webp",
            f"{HEADSHOTS_BASE_URL}/{name_slug}-square.webp"
        )
    # Try partial matches
    for hs in HEADSHOT_SPEAKERS:
        if hs in name_slug or name_slug in hs:
            return (
                f"{HEADSHOTS_BASE_URL}/{hs}-portrait.webp",
                f"{HEADSHOTS_BASE_URL}/{hs}-square.webp"
            )
    return (None, None)


def load_scriptures(conn):
    """Load all scripture JSON files into the database."""
    scriptures_dir = DATA_DIR / "scriptures"
    cursor = conn.cursor()

    scripture_data = []

    for volume_dir in scriptures_dir.iterdir():
        if not volume_dir.is_dir():
            continue

        volume_abbr = volume_dir.name
        print(f"  Loading {volume_abbr}...")

        for json_file in sorted(volume_dir.glob("*.json")):
            with open(json_file) as f:
                data = json.load(f)

            metadata = data.get("metadata", {})
            chapter_content = data.get("chapter_content", {})

            for verse in chapter_content.get("verses", []):
                scripture_data.append((
                    metadata.get("volume", ""),
                    metadata.get("volume_abbr", volume_abbr),
                    metadata.get("book_id", 0),
                    metadata.get("book_name", ""),
                    metadata.get("book_abbr", ""),
                    metadata.get("chapter_number", 0),
                    verse.get("verse_number", 0),
                    verse.get("verse_id", ""),
                    verse.get("verse_text", ""),
                    verse.get("word_count", 0)
                ))

    print(f"  Inserting {len(scripture_data)} verses...")

    execute_values(
        cursor,
        """
        INSERT INTO scriptures
        (volume, volume_abbr, book_id, book_name, book_abbr, chapter_number,
         verse_number, verse_id, verse_text, word_count)
        VALUES %s
        ON CONFLICT (verse_id) DO NOTHING
        """,
        scripture_data,
        page_size=1000
    )

    conn.commit()
    print(f"  Scriptures loaded: {cursor.rowcount} inserted")
    return cursor.rowcount


def load_talks(conn):
    """Load all talk JSON files into the database."""
    talks_dir = DATA_DIR / "talks"
    cursor = conn.cursor()

    # First pass: collect all unique speakers
    speakers = {}
    talks_data = []

    for conf_dir in sorted(talks_dir.iterdir()):
        if not conf_dir.is_dir():
            continue

        conf_name = conf_dir.name
        print(f"  Processing {conf_name}...")

        for json_file in sorted(conf_dir.glob("*.json")):
            with open(json_file) as f:
                data = json.load(f)

            # Handle different JSON formats
            speaker = data.get("speaker", data.get("expected_speaker", "Unknown"))
            # Clean speaker name (remove titles for slug)
            speaker_clean = re.sub(r'^(Elder|President|Sister|Bishop)\s+', '', speaker)
            speaker_slug = slugify(speaker_clean)

            # Use slug as key to dedupe variations of same speaker name
            if speaker_slug not in speakers:
                portrait, square = get_headshot_urls(speaker_slug)
                speakers[speaker_slug] = {
                    "name": speaker,
                    "slug": speaker_slug,
                    "calling": data.get("calling", ""),
                    "portrait": portrait,
                    "square": square
                }

            # Get content from either format
            content = data.get("content", "") or data.get("full_text", "")
            if not content and "paragraphs" in data:
                paragraphs = data["paragraphs"]
                if paragraphs and isinstance(paragraphs[0], str):
                    # Older format: list of strings
                    content = "\n\n".join(paragraphs)
                elif paragraphs and isinstance(paragraphs[0], dict):
                    # Newer format: list of objects with 'text' key
                    content = "\n\n".join(p.get("text", "") for p in paragraphs)

            # Parse conference from folder name (e.g., "2024-A" -> "April 2024")
            conf_match = re.match(r'(\d{4})-([AO])', conf_name)
            if conf_match:
                year, session = conf_match.groups()
                conference = f"{'April' if session == 'A' else 'October'} {year}"
            else:
                conference = data.get("conference", conf_name)

            # Ensure all fields are proper types (no dicts)
            session_info = data.get("session_info", data.get("session", ""))
            if isinstance(session_info, dict):
                session_info = json.dumps(session_info)

            kicker = data.get("kicker", "")
            if isinstance(kicker, dict):
                kicker = json.dumps(kicker)

            talks_data.append({
                "talk_id": str(data.get("talk_id", data.get("hex_id", data.get("id", json_file.stem)))),
                "decimal_id": data.get("decimal_id"),
                "speaker_slug": speaker_slug,
                "title": str(data.get("title", data.get("expected_title", ""))),
                "conference": conference,
                "session": str(session_info) if session_info else "",
                "kicker": str(kicker) if kicker else "",
                "content": content,
                "paragraphs": json.dumps(data.get("paragraphs")) if data.get("paragraphs") else None,
                "source_url": str(data.get("source_url", data.get("api_url", data.get("ajax_url", ""))))
            })

    # Insert speakers
    print(f"  Inserting {len(speakers)} speakers...")
    speaker_ids = {}

    for slug, info in speakers.items():
        cursor.execute(
            """
            INSERT INTO speakers (name, name_slug, calling, headshot_portrait, headshot_square)
            VALUES (%s, %s, %s, %s, %s)
            ON CONFLICT (name_slug) DO UPDATE SET
                calling = COALESCE(NULLIF(EXCLUDED.calling, ''), speakers.calling),
                headshot_portrait = COALESCE(EXCLUDED.headshot_portrait, speakers.headshot_portrait),
                headshot_square = COALESCE(EXCLUDED.headshot_square, speakers.headshot_square)
            RETURNING id
            """,
            (info["name"], info["slug"], info["calling"], info["portrait"], info["square"])
        )
        speaker_ids[slug] = cursor.fetchone()[0]

    conn.commit()

    # Insert talks
    print(f"  Inserting {len(talks_data)} talks...")

    for talk in talks_data:
        cursor.execute(
            """
            INSERT INTO talks
            (talk_id, decimal_id, speaker_id, title, conference, session, kicker, content, paragraphs, source_url)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            ON CONFLICT (talk_id) DO NOTHING
            """,
            (
                talk["talk_id"],
                talk["decimal_id"],
                speaker_ids.get(talk["speaker_slug"]),
                talk["title"],
                talk["conference"],
                talk["session"],
                talk["kicker"],
                talk["content"],
                talk["paragraphs"],
                talk["source_url"]
            )
        )

    conn.commit()
    print(f"  Talks loaded: {len(talks_data)} processed")
    return len(talks_data)


def main():
    print("Connecting to Cloud SQL...")
    conn = psycopg2.connect(
        host=DB_HOST,
        database=DB_NAME,
        user=DB_USER,
        password=DB_PASS
    )

    try:
        print("\n1. Loading scriptures...")
        scripture_count = load_scriptures(conn)

        print("\n2. Loading talks...")
        talk_count = load_talks(conn)

        # Print summary
        cursor = conn.cursor()
        cursor.execute("SELECT COUNT(*) FROM scriptures")
        total_verses = cursor.fetchone()[0]

        cursor.execute("SELECT COUNT(*) FROM talks")
        total_talks = cursor.fetchone()[0]

        cursor.execute("SELECT COUNT(*) FROM speakers")
        total_speakers = cursor.fetchone()[0]

        cursor.execute("SELECT COUNT(*) FROM speakers WHERE headshot_square IS NOT NULL")
        speakers_with_headshots = cursor.fetchone()[0]

        print("\n" + "="*50)
        print("LOAD COMPLETE")
        print("="*50)
        print(f"Scriptures (verses): {total_verses:,}")
        print(f"Talks:               {total_talks:,}")
        print(f"Speakers:            {total_speakers:,}")
        print(f"  with headshots:    {speakers_with_headshots:,}")
        print("="*50)

    finally:
        conn.close()


if __name__ == "__main__":
    main()
