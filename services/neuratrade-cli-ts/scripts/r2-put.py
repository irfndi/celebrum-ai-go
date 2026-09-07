"""Minimal S3 SigV4 PUT for Cloudflare R2 (stdlib only, no boto3).

Usage: r2-put.py <bucket> <key> <file> [--gzip]
Env: R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY
"""
import gzip
import hashlib
import hmac
import os
import sys
import urllib.request
from datetime import datetime, timezone


def sign(key: bytes, msg: str) -> bytes:
    return hmac.new(key, msg.encode(), hashlib.sha256).digest()


def main() -> None:
    bucket, key, path = sys.argv[1:4]
    # ponytail: gzip at rest+wire; 240MB/day ledger -> ~30MB upload.
    body = open(path, "rb").read()
    if "--gzip" in sys.argv:
        body = gzip.compress(body)
    acct = os.environ["R2_ACCOUNT_ID"]
    ak = os.environ["R2_ACCESS_KEY_ID"]
    sk = os.environ["R2_SECRET_ACCESS_KEY"]
    host = f"{acct}.r2.cloudflarestorage.com"
    uri = f"/{bucket}/{key}"
    now = datetime.now(timezone.utc)
    amz = now.strftime("%Y%m%dT%H%M%SZ")
    short = now.strftime("%Y%m%d")
    payload_hash = hashlib.sha256(body).hexdigest()
    headers = {
        "host": host,
        "x-amz-content-sha256": payload_hash,
        "x-amz-date": amz,
    }
    signed = ";".join(sorted(headers))
    canonical = "\n".join(
        [
            "PUT",
            uri,
            "",
            "".join(f"{k}:{headers[k]}\n" for k in sorted(headers)),
            signed,
            payload_hash,
        ]
    )
    scope = f"{short}/auto/s3/aws4_request"
    to_sign = "\n".join(
        ["AWS4-HMAC-SHA256", amz, scope, hashlib.sha256(canonical.encode()).hexdigest()]
    )
    k = sign(f"AWS4{sk}".encode(), short)
    k = sign(k, "auto")
    k = sign(k, "s3")
    k = sign(k, "aws4_request")
    sig = hmac.new(k, to_sign.encode(), hashlib.sha256).hexdigest()
    req = urllib.request.Request(
        f"https://{host}{uri}",
        data=body,
        method="PUT",
        headers={
            **{k: v for k, v in headers.items() if k != "host"},
            "Authorization": (
                f"AWS4-HMAC-SHA256 Credential={ak}/{scope}, "
                f"SignedHeaders={signed}, Signature={sig}"
            ),
        },
    )
    if "--gzip" in sys.argv:
        req.add_header("Content-Encoding", "gzip")
    with urllib.request.urlopen(req, timeout=600) as r:
        print(f"PUT {bucket}/{key} -> {r.status} ({len(body)} bytes wire)")


main()
