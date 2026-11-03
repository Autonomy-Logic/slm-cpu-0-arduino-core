#!/usr/bin/env python3
"""Repoint one package's artefacts in an index at a locally served copy.

Lets CI install the archives it just built before any release exists.
Third-party entries keep their URLs. arduino-cli only accepts http(s) for
assets, so the base must be an HTTP server, not a file:// path.

  tools/ci/localize-index.py <index.json> <base-url> [--package FACTS]
"""
import argparse
import json
import sys


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("index")
    ap.add_argument("base")
    ap.add_argument("--package", default="FACTS")
    args = ap.parse_args()

    if not args.base.startswith(("http://", "https://")):
        sys.exit("base must be http(s): arduino-cli rejects file:// for assets")

    base = args.base.rstrip("/")
    with open(args.index, encoding="utf-8") as f:
        idx = json.load(f)

    n = 0
    for pkg in idx["packages"]:
        if pkg["name"] != args.package:
            continue
        for plat in pkg.get("platforms", []):
            if plat.get("archiveFileName"):
                plat["url"] = f"{base}/{plat['archiveFileName']}"
                n += 1
        for tool in pkg.get("tools", []):
            for s in tool.get("systems", []):
                if s["url"].startswith("https://github.com/Autonomy-Logic/"):
                    s["url"] = f"{base}/{s['archiveFileName']}"
                    n += 1

    if n == 0:
        sys.exit(f"no artefacts found for package {args.package!r}")
    with open(args.index, "w", encoding="utf-8") as f:
        json.dump(idx, f, indent=2)
        f.write("\n")
    print(f"repointed {n} url(s) in {args.package} at {base}")


if __name__ == "__main__":
    main()
