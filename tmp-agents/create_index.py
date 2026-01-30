#!/usr/bin/env python3
"""
Create an index JSON file for all October 2021 talks
"""

import json
from pathlib import Path

OUTPUT_DIR = Path("/Users/justinjones/Developer/temple-square/tmp/data/talks/2021-O")

# Collect all talk metadata
talks_index = []

for json_file in sorted(OUTPUT_DIR.glob("*.json")):
    with open(json_file, 'r', encoding='utf-8') as f:
        talk_data = json.load(f)

    # Create index entry with essential metadata
    index_entry = {
        "talk_id": talk_data["talk_id"],
        "title": talk_data["title"],
        "speaker": talk_data["speaker"],
        "calling": talk_data["calling"],
        "word_count": talk_data["word_count"],
        "file": f"{talk_data['talk_id']}.json"
    }

    talks_index.append(index_entry)

# Create master index file
index_data = {
    "conference": "October 2021 General Conference",
    "date": "October 2021",
    "total_talks": len(talks_index),
    "total_words": sum(t["word_count"] for t in talks_index),
    "extraction_date": "2026-01-27",
    "extraction_method": "AJAX API",
    "talks": talks_index
}

# Save index
index_file = OUTPUT_DIR / "index.json"
with open(index_file, 'w', encoding='utf-8') as f:
    json.dump(index_data, f, indent=2, ensure_ascii=False)

print(f"Index created: {index_file}")
print(f"Total talks: {len(talks_index)}")
print(f"Total words: {index_data['total_words']:,}")
