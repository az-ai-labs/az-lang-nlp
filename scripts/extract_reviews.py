#!/usr/bin/env python3
"""Extract and clean Azerbaijani review sentiment data from Arrow dataset files.

Usage:
    python scripts/extract_reviews.py

Reads train and test Arrow files from:
    az_data/hajili_azerbaijani_review_sentiment_classification/

Writes cleaned output to:
    az_data/reviews_clean.tsv  (tab-separated, columns: content<TAB>score, no header)

Cleaning steps applied:
    1. Deduplicate by content (keep first occurrence across both splits)
    2. Drop rows where len(content.strip()) < 10
    3. Remove texts that appear with multiple different scores (ambiguous sentiment)
"""

import sys
from collections import defaultdict
from pathlib import Path

import pyarrow.ipc as ipc


SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent

TRAIN_ARROW = PROJECT_ROOT / "az_data" / "hajili_azerbaijani_review_sentiment_classification" / "train" / "data-00000-of-00001.arrow"
TEST_ARROW  = PROJECT_ROOT / "az_data" / "hajili_azerbaijani_review_sentiment_classification" / "test"  / "data-00000-of-00001.arrow"
OUTPUT_TSV  = PROJECT_ROOT / "az_data" / "reviews_clean.tsv"

MIN_CONTENT_LEN = 10


def read_arrow(path: Path) -> list[dict]:
    """Read an Arrow IPC streaming file and return a list of dicts with 'content' and 'score'."""
    records = []
    with path.open("rb") as f:
        reader = ipc.open_stream(f)
        for batch in reader:
            contents = batch.column("content").to_pylist()
            scores   = batch.column("score").to_pylist()
            for content, score in zip(contents, scores):
                records.append({"content": content, "score": score})
    return records


def clean(records: list[dict]) -> list[dict]:
    """Apply deduplication, length filter, and ambiguous-score removal."""
    # Drop rows with None content, then deduplicate — keep first occurrence
    seen_content: set[str] = set()
    deduped = []
    for row in records:
        c = row["content"]
        if c is None:
            continue
        if c not in seen_content:
            seen_content.add(c)
            deduped.append(row)

    # Drop short content
    length_filtered = [r for r in deduped if len(r["content"].strip()) >= MIN_CONTENT_LEN]

    # Find texts that map to more than one score
    content_scores: dict[str, set] = defaultdict(set)
    for row in length_filtered:
        content_scores[row["content"]].add(row["score"])
    ambiguous = {c for c, scores in content_scores.items() if len(scores) > 1}

    cleaned = [r for r in length_filtered if r["content"] not in ambiguous]
    return cleaned


def main() -> None:
    for path in (TRAIN_ARROW, TEST_ARROW):
        if not path.exists():
            print(f"ERROR: Arrow file not found: {path}", file=sys.stderr)
            sys.exit(1)

    print("Reading Arrow files...")
    train_records = read_arrow(TRAIN_ARROW)
    test_records  = read_arrow(TEST_ARROW)

    train_count = len(train_records)
    test_count  = len(test_records)
    combined    = train_records + test_records
    total_before = len(combined)

    print(f"\nRows before cleaning:")
    print(f"  train : {train_count}")
    print(f"  test  : {test_count}")
    print(f"  total : {total_before}")

    cleaned = clean(combined)
    total_after = len(cleaned)

    score_counts: dict[int, int] = defaultdict(int)
    for row in cleaned:
        score_counts[row["score"]] += 1

    print(f"\nRows per score after cleaning:")
    for star in sorted(score_counts):
        print(f"  score {star}: {score_counts[star]}")

    print(f"\nTotal rows after cleaning : {total_after}")
    print(f"Rows removed              : {total_before - total_after}")

    OUTPUT_TSV.parent.mkdir(parents=True, exist_ok=True)
    with OUTPUT_TSV.open("w", encoding="utf-8") as out:
        for row in cleaned:
            content = row["content"].replace("\t", " ").replace("\n", " ").replace("\r", " ")
            out.write(f"{content}\t{row['score']}\n")

    print(f"\nOutput written to: {OUTPUT_TSV}")


if __name__ == "__main__":
    main()
