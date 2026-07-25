#!/usr/bin/env python3
"""Reflexion-style Actor → Critic → Propose loop over Starling Lite MCP (HTTP)."""

from __future__ import annotations

import argparse
import json
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path


def rpc(url: str, method: str, params: dict | None = None, id_: int = 1) -> dict:
    body = {"jsonrpc": "2.0", "id": id_, "method": method}
    if params is not None:
        body["params"] = params
    req = urllib.request.Request(
        url,
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        return json.loads(resp.read().decode())


def tool_text(result: dict) -> tuple[str, bool]:
    r = result.get("result") or {}
    if result.get("error"):
        return json.dumps(result["error"]), True
    parts = []
    for c in r.get("content") or []:
        if c.get("text"):
            parts.append(c["text"])
    return "\n".join(parts), bool(r.get("isError"))


def call_tool(url: str, name: str, arguments: dict, id_: int) -> tuple[str, bool]:
    res = rpc(url, "tools/call", {"name": name, "arguments": arguments}, id_=id_)
    return tool_text(res)


def critic(scan: dict) -> tuple[bool, str]:
    """Return (approved, reason). Approved means continue to propose (not apply)."""
    summary = scan.get("summary") or {}
    crit = int(summary.get("critical") or 0)
    high = int(summary.get("high") or 0)
    score = (scan.get("score") or {}).get("overall")
    grade = (scan.get("score") or {}).get("grade")
    if crit >= 5:
        return False, f"blast radius too high: {crit} critical findings — human review required"
    if score is not None and float(score) < 10:
        return False, f"score {score} grade {grade} — refuse auto-heal proposals without human"
    return True, f"ok to propose remediations (critical={crit} high={high} grade={grade})"


def main() -> int:
    ap = argparse.ArgumentParser(description="Reflexion agent over Starling Lite MCP")
    ap.add_argument("--mcp", default="http://127.0.0.1:8822/mcp", help="MCP HTTP endpoint")
    ap.add_argument("--path", required=True, help="Manifest path to scan")
    ap.add_argument("--trace-dir", default=None, help="Trace output dir")
    args = ap.parse_args()

    root = Path(__file__).resolve().parents[2]
    path = Path(args.path)
    if not path.is_absolute():
        path = (Path.cwd() / path).resolve()
    if not path.exists():
        # try relative to army root
        alt = root / args.path
        if alt.exists():
            path = alt
        else:
            print(f"error: path not found: {args.path}", file=sys.stderr)
            return 1

    trace_dir = Path(args.trace_dir) if args.trace_dir else Path(__file__).parent / "traces"
    trace_dir.mkdir(parents=True, exist_ok=True)
    trace_path = trace_dir / f"trace-{int(time.time())}.jsonl"

    def audit(event: str, payload: dict) -> None:
        row = {"ts": time.time(), "event": event, **payload}
        with trace_path.open("a") as f:
            f.write(json.dumps(row) + "\n")
        print(f"[{event}] {json.dumps(payload)[:200]}...")

    print(f"MCP: {args.mcp}")
    print(f"path: {path}")
    print(f"trace: {trace_path}")

    try:
        init = rpc(args.mcp, "initialize", {
            "protocolVersion": "2024-11-05",
            "clientInfo": {"name": "reflexion-agent", "version": "0.1.0"},
            "capabilities": {},
        })
        audit("initialize", {"server": (init.get("result") or {}).get("serverInfo")})
    except urllib.error.URLError as e:
        print(f"error: cannot reach MCP at {args.mcp}: {e}", file=sys.stderr)
        print("start: ./bin/starling-lite --transport=http --addr=:8822", file=sys.stderr)
        return 2

    # Actor: scan
    text, err = call_tool(args.mcp, "scan_manifest", {"path": str(path), "source": path.name}, id_=2)
    audit("actor.scan_manifest", {"is_error": err, "bytes": len(text)})
    if err:
        print("scan error:", text)
        return 3

    try:
        scan = json.loads(text)
    except json.JSONDecodeError:
        print(text)
        return 3

    score = scan.get("score") or {}
    summary = scan.get("summary") or {}
    print("\n=== SCAN ===")
    print(f"source: {scan.get('source')}  score={score.get('overall')} grade={score.get('grade')}")
    print(f"summary: {summary}")

    # Critic
    ok, reason = critic(scan)
    audit("critic", {"approved": ok, "reason": reason})
    print(f"\n=== CRITIC ===\n{reason}")
    if not ok:
        print("stopped before propose.")
        return 4

    # Healer: propose for top findings
    findings = scan.get("findings") or []
    proposals = []
    for i, f in enumerate(findings[:5]):
        issue = f.get("id") or f.get("message") or "unknown"
        resource = f.get("resource") or "Deployment/unknown"
        kind, name = "Deployment", "unknown"
        if "/" in resource:
            kind, name = resource.split("/", 1)
        text, err = call_tool(
            args.mcp,
            "propose_remediation",
            {"kind": kind, "name": name, "namespace": "default", "issue": issue},
            id_=10 + i,
        )
        audit("healer.propose", {"issue": issue, "resource": resource, "is_error": err})
        if not err:
            try:
                proposals.append(json.loads(text))
            except json.JSONDecodeError:
                proposals.append({"raw": text})

    print(f"\n=== PROPOSALS ({len(proposals)}) ===")
    for p in proposals:
        print(json.dumps(p, indent=2)[:500])
        print("---")

    audit("done", {"proposals": len(proposals), "trace": str(trace_path)})
    print(f"\ntrace written: {trace_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
