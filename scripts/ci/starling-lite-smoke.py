#!/usr/bin/env python3
"""Smoke-test a running Starling Lite MCP over HTTP (stdlib only).

Used by .github/workflows/deploy-starling-lite.yml against the freshly built
container and against production after `railway up`.

  scripts/ci/starling-lite-smoke.py --url https://mcp.warbleoss.org \
      [--revision <git sha>] [--wait 480] [--manifest path/to/bad.yaml]

Checks:
  1. GET /health is ok (and, with --revision, reports that exact revision;
     polls up to --wait seconds so a rolling deploy can finish)
  2. tools/list returns the scan + remediation tools and no write tools
  3. scan_manifest on an intentionally bad manifest is complete
     (incomplete == false), graded (not N/A) and has high findings
  4. propose_remediation with the manifest names the real container
     (no PLACEHOLDER) and stays dry-run
"""
import argparse
import json
import sys
import time
import urllib.error
import urllib.request

PROTO = "2026-07-28"


def http(url, body=None, timeout=60):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method="POST" if data else "GET")
    req.add_header("User-Agent", "starling-lite-smoke")
    if data:
        req.add_header("Content-Type", "application/json")
        req.add_header("MCP-Protocol-Version", PROTO)
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return json.loads(r.read().decode())


def rpc(mcp, rid, method, params=None):
    msg = {"jsonrpc": "2.0", "id": rid, "method": method}
    if params is not None:
        msg["params"] = params
    r = http(mcp, msg)
    if r.get("error"):
        fail(f"{method}: {r['error']}")
    return r["result"]


def call_tool(mcp, rid, name, args):
    res = rpc(mcp, rid, "tools/call", {"name": name, "arguments": args})
    if res.get("isError"):
        fail(f"{name} returned isError: {res}")
    return json.loads(res["content"][0]["text"])


def fail(msg):
    print(f"::error::{msg}")
    sys.exit(1)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--url", required=True)
    ap.add_argument("--revision", default="")
    ap.add_argument("--wait", type=int, default=120, help="seconds to wait for health/revision")
    ap.add_argument("--manifest", default="shrikeops/scanner/internal/engine/testdata/manifests/bad/privileged-pod.yaml")
    a = ap.parse_args()
    base = a.url.rstrip("/")
    mcp = base + "/mcp"

    # 1. health (+ revision)
    deadline = time.time() + a.wait
    health, last = None, "no response"
    while True:
        try:
            health = http(f"{base}/health?ts={int(time.time())}", timeout=15)
            last = f"status={health.get('status')} revision={health.get('revision')}"
            if health.get("status") == "ok" and (not a.revision or health.get("revision") == a.revision):
                break
        except (urllib.error.URLError, OSError, ValueError) as e:
            last = str(e)
        if time.time() > deadline:
            want = f" with revision {a.revision}" if a.revision else ""
            fail(f"{base}/health not ok{want} after {a.wait}s (last: {last})")
        print(f"waiting for {base}/health ({last})", flush=True)
        time.sleep(10)
    print("ok health", json.dumps({k: health.get(k) for k in ("server", "version", "revision", "mode", "status")}))
    if health.get("mode") != "readonly":
        fail(f"public server must be readonly, got mode={health.get('mode')}")

    # 2. tools/list
    tools = rpc(mcp, 1, "tools/list")["tools"]
    names = {t["name"] for t in tools}
    for need in ("scan_manifest", "propose_remediation", "cluster_summary"):
        if need not in names:
            fail(f"tools/list missing {need}: {sorted(names)}")
    if "scale_deployment" in names:
        fail("write tool scale_deployment exposed in readonly mode")
    print(f"ok tools/list count={len(tools)}")

    # 3. scan_manifest must be complete and graded
    manifest = open(a.manifest).read()
    rep = call_tool(mcp, 2, "scan_manifest", {"manifest": manifest, "source": "smoke-bad.yaml"})
    score = rep.get("score") or {}
    if rep.get("incomplete") is not False:
        fail(f"scan_manifest incomplete={rep.get('incomplete')!r} warning={rep.get('warning')!r} (scanners missing in image?)")
    if score.get("grade") in (None, "N/A"):
        fail(f"scan_manifest not graded: {score}")
    highs = [f for f in rep.get("findings") or [] if f.get("severity") in ("critical", "high")]
    if not highs:
        fail(f"bad manifest produced no high/critical findings: score={score}")
    print(f"ok scan_manifest grade={score.get('grade')} overall={score.get('overall')} high+critical={len(highs)}")

    # 4. propose_remediation names the real container
    import re
    m = re.search(r"^kind:\s*(\S+)", manifest, re.M)
    n = re.search(r"^metadata:\s*\n(?:\s+.*\n)*?\s+name:\s*(\S+)", manifest, re.M)
    kind, name = (m.group(1) if m else "Pod"), (n.group(1) if n else "smoke")
    prop = call_tool(mcp, 3, "propose_remediation", {
        "kind": kind, "name": name, "namespace": "default",
        "issue": "missing-resource-limits", "manifest": manifest,
    })
    patch = json.dumps(prop.get("patch"))
    if "PLACEHOLDER" in patch:
        fail(f"propose_remediation still emits PLACEHOLDER: {prop.get('patch')}")
    if prop.get("dry_run") is not True:
        fail(f"propose_remediation must be dry_run=true, got {prop.get('dry_run')!r}")
    print(f"ok propose_remediation {kind}/{name} patch_type={prop.get('patch_type')} containers={prop.get('containers')}")
    print("smoke OK")


if __name__ == "__main__":
    main()
