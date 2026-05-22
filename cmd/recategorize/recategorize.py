#!/usr/bin/env python3
"""Re-categorize existing places.json using updated category_rules.json.
Only touches places that currently have category "Sonstiges" or null."""

import json, re, sys
from pathlib import Path

# Load rules
with open("internal/mapsreview/data/category_rules.json") as f:
    rules = json.load(f)

# Pre-process: build blocked set, lowercased keyword -> category map
blocked = set(s.lower() for s in rules["blocked_strings"])
name_keywords = set(s.lower() for s in rules["name_keywords"])

# Categories sorted by priority (first match wins)
categories = rules["categories"]

def strip_private_use(s):
    result = []
    for ch in s:
        cp = ord(ch)
        if 0xE000 <= cp <= 0xF8FF:
            continue
        if 0x2500 <= cp <= 0x257F:
            continue
        result.append(ch)
    return "".join(result)

def is_review_snippet(lower):
    count = 0
    for kw in rules["review_keywords"]:
        if kw in lower:
            count += 1
    return count >= 2

def is_business_name(candidate):
    if "|" in candidate:
        return True
    if re.search(r"(?i)\bimbiss\b|\bdöner\b.*&|&.*\bdöner\b", candidate):
        if not re.search(r"(?i)restaurant$|imbiss$", candidate):
            return True
    return False

def clean_category_candidate(value, name=""):
    candidate = value.strip()
    if not candidate:
        return ""
    # Strip Google Maps UI decorations
    candidate = re.sub(r'[·•]€?€?[·•]?', '', candidate)
    candidate = strip_private_use(candidate).strip()
    # Strip leading/trailing non-letter/number (keep periods for abbreviations like e.V.)
    candidate = candidate.strip(',;:!?@#$%^&*()_+-=[]{}|\\\'\"`~<>/ \t\n\r')
    candidate = candidate.strip()
    if not candidate:
        return ""
    lower = candidate.lower()
    if name and lower == name.lower().strip():
        return ""
    # Reject URLs
    if re.search(r'(?i)\.(de|com|net|org|io|info)\b|https?://|www\.', candidate):
        return ""
    # Reject long strings
    if len(candidate) > 50:
        return ""
    # Reject review snippets
    if is_review_snippet(lower):
        return ""
    # Reject business names
    if is_business_name(candidate):
        return ""
    # Blocked strings
    if lower in blocked:
        return ""
    # Reject ratings, codes, etc.
    if re.search(r'(?i)^[1-5](?:[,.][0-9])?$|^\(?[0-9][0-9.]*\)?$|€|geöffnet|geschlossen|adresse|telefon|\.de\b|\b9\d{4}\b', candidate):
        return ""
    # Must contain at least one letter
    if not any(ch.isalpha() for ch in candidate):
        return ""
    return candidate

def normalize_category(candidate):
    if not candidate:
        return ""
    lower = candidate.lower()
    for bucket in categories:
        if len(bucket["keywords"]) == 0:
            return bucket["name"]  # catch-all
        for kw in bucket["keywords"]:
            if kw in lower:
                return bucket["name"]
    return categories[-1]["name"]

def parent_of(category_name):
    """Return the parent group for a category name."""
    for bucket in categories:
        if bucket["name"] == category_name:
            return bucket.get("parent", bucket["name"])
    return ""

def infer_from_name(name):
    if not name:
        return ""
    lower = name.lower()
    has_kw = any(kw in lower for kw in name_keywords)
    if not has_kw:
        return ""
    candidate = clean_category_candidate(name, "")
    if not candidate:
        candidate = name
    return normalize_category(candidate)

# Load places
with open("output/places.json") as f:
    places = json.load(f)

changed = 0
stats = {}

for place in places:
    old_cat = place.get("category")
    
    # Always compute and set parentCategory
    cat_for_parent = old_cat or ""
    parent = parent_of(cat_for_parent)
    if parent:
        place["parentCategory"] = parent
    else:
        place.pop("parentCategory", None)
    
    # Only recategorize if currently Sonstiges
    if old_cat not in (None, "Sonstiges"):
        continue
    
    name = place.get("name", "")
    new_cat = infer_from_name(name)
    
    if new_cat and new_cat != old_cat and new_cat != "Sonstiges":
        place["category"] = new_cat
        changed += 1
        stats[new_cat] = stats.get(new_cat, 0) + 1
        # Update parentCategory for recategorized place
        parent = parent_of(new_cat)
        if parent:
            place["parentCategory"] = parent

# Write back
with open("output/places.json", "w") as f:
    json.dump(places, f, ensure_ascii=False, indent=2)

print(f"Recategorized {changed} places:")
for cat, cnt in sorted(stats.items(), key=lambda x: -x[1]):
    print(f"  {cat}: {cnt}")

# Also update CSV
if changed > 0:
    import csv, io
    fieldnames = [
        "id", "name", "postcode", "address", "rating", "reviewCount",
        "category", "parentCategory", "lat", "lng", "bezirkId", "bezirkName",
        "hasDefamationNotice", "removedMin", "removedMax", "removedEstimate",
        "deletionRatioPct", "realRatingAdjusted", "removedText",
        "url", "readAt", "placeState", "status", "error"
    ]
    with open("output/places.csv", "w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=fieldnames, extrasaction="ignore")
        w.writeheader()
        for place in places:
            row = {k: place.get(k) for k in fieldnames}
            row["hasDefamationNotice"] = str(place.get("hasDefamationNotice", False))
            w.writerow(row)
    print(f"\nCSV updated.")
