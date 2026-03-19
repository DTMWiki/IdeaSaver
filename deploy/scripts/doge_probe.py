#!/usr/bin/env python3
"""
Probe DogeCloud auth API for temporary credentials and upload metadata.

Usage:
  python deploy/scripts/doge_probe.py \
    --access-id "<DOGE_ACCESS_ID>" \
    --secret-key "<DOGE_SECRET_KEY>"
"""

from __future__ import annotations

import argparse
import hashlib
import hmac
import json
import sys
from urllib.parse import urlparse
import urllib.error
import urllib.request
from typing import Any, Dict, Iterable


def sign(secret_key: str, request_uri: str, body: str) -> str:
    sign_str = f"{request_uri}\n{body}".encode("utf-8")
    digest = hmac.new(secret_key.encode("utf-8"), sign_str, hashlib.sha1).hexdigest()
    return digest


def call_api(api_base: str, access_id: str, secret_key: str, path: str, payload: Dict[str, Any]) -> Dict[str, Any]:
    body = json.dumps(payload, separators=(",", ":"), ensure_ascii=False)
    token = f"TOKEN {access_id}:{sign(secret_key, path, body)}"
    req = urllib.request.Request(
        url=f"{api_base}{path}",
        data=body.encode("utf-8"),
        method="POST",
        headers={
            "Authorization": token,
            "Content-Type": "application/json",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw = resp.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"HTTP {e.code}: {raw}") from e
    except urllib.error.URLError as e:
        raise RuntimeError(f"Network error: {e}") from e

    try:
        return json.loads(raw)
    except json.JSONDecodeError as e:
        raise RuntimeError(f"Invalid JSON response: {raw}") from e


def pick_first(data: Dict[str, Any], keys: Iterable[str]) -> Any:
    for k in keys:
        if k in data and data[k] not in (None, "", []):
            return data[k]
    return None


def derive_region(endpoint: str) -> str:
    text = endpoint.strip()
    if not text:
        return ""
    if "://" not in text:
        text = "https://" + text
    host = urlparse(text).hostname or ""
    if not host:
        return ""
    for part in host.split("."):
        if part.startswith(("ap-", "cn-", "us-", "eu-")):
            return part
    parts = host.split(".")
    if len(parts) >= 2 and parts[0] in ("cos", "s3"):
        return parts[1]
    return ""


def select_bucket(data: Dict[str, Any], preferred_bucket: str) -> Dict[str, Any] | None:
    buckets = data.get("Buckets")
    if not isinstance(buckets, list) or len(buckets) == 0:
        return None

    pref = preferred_bucket.strip()
    if pref:
        for item in buckets:
            if not isinstance(item, dict):
                continue
            name = str(item.get("name", "")).strip()
            s3_bucket = str(item.get("s3Bucket", "")).strip()
            if name == pref or s3_bucket == pref:
                return item
    for item in buckets:
        if isinstance(item, dict):
            return item
    return None


def summarize(label: str, resp: Dict[str, Any], preferred_bucket: str) -> None:
    print(f"\n=== {label} ===")
    print(json.dumps(resp, ensure_ascii=False, indent=2))

    data = resp.get("data")
    if not isinstance(data, dict):
        return

    endpoint = pick_first(data, ["s3Endpoint", "S3Endpoint", "Endpoint", "endpoint", "OSS_Endpoint", "ossEndpoint"])
    bucket = pick_first(data, ["s3Bucket", "S3Bucket", "Bucket", "bucket", "OSS_Bucket", "ossBucket"])
    vod_upload_info = pick_first(data, ["VodUploadInfo", "vodUploadInfo"])
    credentials = pick_first(data, ["Credentials", "credentials"])
    region = pick_first(data, ["s3Region", "S3Region", "Region", "region"])
    selected_bucket = select_bucket(data, preferred_bucket)

    if isinstance(selected_bucket, dict):
        endpoint = selected_bucket.get("s3Endpoint") or endpoint
        bucket = selected_bucket.get("s3Bucket") or bucket
        print(f"[Extracted] selected bucket entry: name={selected_bucket.get('name')} s3Bucket={selected_bucket.get('s3Bucket')}")

    if endpoint is not None:
        print(f"[Extracted] endpoint: {endpoint}")
    if bucket is not None:
        print(f"[Extracted] bucket: {bucket}")
    if not region and isinstance(endpoint, str):
        region = derive_region(endpoint)
    if region is not None and region != "":
        print(f"[Extracted] region: {region}")
    if vod_upload_info is not None:
        print(f"[Extracted] VodUploadInfo: {json.dumps(vod_upload_info, ensure_ascii=False)}")

    if isinstance(credentials, dict):
        ak = credentials.get("accessKeyId")
        sk = credentials.get("secretAccessKey")
        st = credentials.get("sessionToken")
        exp = credentials.get("expiredTime") or credentials.get("expiration")
        if ak and sk and st:
            print("[Extracted] temporary credentials:")
            print(f"  accessKeyId={ak}")
            print(f"  secretAccessKey={sk}")
            print(f"  sessionToken={st}")
            if exp:
                print(f"  expires={exp}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Probe DogeCloud tmp token and upload metadata.")
    parser.add_argument("--access-id", required=True, help="DogeCloud AccessID")
    parser.add_argument("--secret-key", required=True, help="DogeCloud Key/Secret")
    parser.add_argument("--api-base", default="https://api.dogecloud.com", help="API base URL")
    parser.add_argument("--bucket-name", default="dtm-ideasaver", help="Preferred OSS bucket name to select from Buckets list")
    parser.add_argument("--vod-name", default="ideasaver-probe.mp4", help="Video name used for VOD_UPLOAD token probe")
    args = parser.parse_args()

    path = "/auth/tmp_token.json"
    payloads = [
        ("OSS_FULL", {"channel": "OSS_FULL", "scopes": ["*"]}),
        (
            "VOD_UPLOAD",
            {
                "channel": "VOD_UPLOAD",
                "vodConfig": {
                    "filename": args.vod_name,
                    "vn": "IdeaSaver Probe",
                },
            },
        ),
    ]

    failed = False
    for label, payload in payloads:
        try:
            resp = call_api(args.api_base.rstrip("/"), args.access_id, args.secret_key, path, payload)
        except Exception as e:  # noqa: BLE001
            failed = True
            print(f"\n=== {label} ===")
            print(f"[ERROR] {e}")
            continue
        summarize(label, resp, args.bucket_name)

    if failed:
        print("\nOne or more API probes failed. Verify AccessID/Key and account permissions.")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
