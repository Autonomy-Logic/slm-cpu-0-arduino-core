#!/usr/bin/env python3
"""Verify every artefact an index names: it resolves, and its size and
SHA-256 match the index. Reports every discrepancy, not only the first.

  tools/ci/verify-index.py package_FACTS_slm_cpu_0_index.json [--only-ours]
"""
import argparse
import hashlib
import json
import sys
import urllib.request

OURS = "https://github.com/Autonomy-Logic/"


def check(name, url, want_sha, want_size, errors):
    try:
        with urllib.request.urlopen(url, timeout=120) as r:
            data = r.read()
    except Exception as e:
        errors.append(f"{name}: cannot fetch {url} -- {e}")
        return
    got_size = len(data)
    if want_size and str(got_size) != str(want_size):
        errors.append(f"{name}: size {got_size}, index says {want_size}")
    algo, _, digest = want_sha.partition(":")
    if algo != "SHA-256":
        errors.append(f"{name}: unexpected checksum algorithm {algo!r}")
        return
    got = hashlib.sha256(data).hexdigest()
    if got.lower() != digest.lower():
        errors.append(f"{name}: sha256 {got}, index says {digest}")
    else:
        print(f"  ok  {name}  ({got_size} bytes)")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("index")
    ap.add_argument("--only-ours", action="store_true",
                    help="check only artefacts hosted by Autonomy-Logic, not the third-party toolchain")
    args = ap.parse_args()

    with open(args.index, encoding="utf-8") as f:
        idx = json.load(f)
    errors = []
    for pkg in idx["packages"]:
        entries = [(f"{pkg['name']} platform {p.get('version', '?')}", p) for p in pkg.get("platforms", [])]
        entries += [(f"{pkg['name']} {t['name']} {t['version']} {s['host']}", s)
                    for t in pkg.get("tools", []) for s in t.get("systems", [])]
        for name, e in entries:
            if not e.get("url"):
                errors.append(f"{name}: empty url in a published index")
                continue
            if args.only_ours and not e["url"].startswith(OURS):
                continue
            check(name, e["url"], e["checksum"], e.get("size"), errors)
    if errors:
        print("\nFAILED:", file=sys.stderr)
        for e in errors:
            print("  " + e, file=sys.stderr)
        sys.exit(1)
    print("\nindex verified")


if __name__ == "__main__":
    main()
