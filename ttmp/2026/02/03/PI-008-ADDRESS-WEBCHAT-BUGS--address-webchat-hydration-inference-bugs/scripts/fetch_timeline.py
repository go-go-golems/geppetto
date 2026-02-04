#!/usr/bin/env python3
import argparse
import json
import sys
import urllib.request

DEFAULT_HEADERS = {
    "User-Agent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0",
    "Accept": "*/*",
    "Accept-Language": "en-US,en;q=0.9",
    "Sec-Fetch-Dest": "empty",
    "Sec-Fetch-Mode": "cors",
    "Sec-Fetch-Site": "same-origin",
    "Priority": "u=4",
}


def fetch(url: str, referrer: str | None) -> bytes:
    headers = dict(DEFAULT_HEADERS)
    if referrer:
        headers["Referer"] = referrer
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req) as resp:
        return resp.read()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--conv-id", required=True)
    ap.add_argument("--base", default="http://localhost:8080")
    ap.add_argument("--out", default="-")
    args = ap.parse_args()

    url = f"{args.base}/timeline?conv_id={args.conv_id}"
    referrer = f"{args.base}/?conv_id={args.conv_id}"
    try:
        data = fetch(url, referrer)
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1

    if args.out == "-":
        sys.stdout.buffer.write(data)
        return 0

    with open(args.out, "wb") as f:
        f.write(data)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
