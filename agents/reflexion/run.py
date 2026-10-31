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


WORKLOAD_KINDS = {
    "Pod", "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet",
    "ReplicationController", "Job", "CronJob",
}


def parse_resource(resource: str) -> tuple[str, str, str, str]:
    """Split a ShrikeOps finding resource into (kind, namespace, name, container).

    Handles "Kind/ns/name", "Kind/ns/name/container:c" (Polaris) and
    "Kind/group/version/ns/name" (kube-score).
    """
    parts = [p for p in (resource or "").split("/") if p]
    container = ""
    if parts and parts[-1].startswith("container:"):
        container = parts.pop()[len("container:"):]
    if not parts:
        return "Deployment", "default", "unknown", container
    kind = parts[0]
    rest = parts[1:]
    if len(rest) >= 3:  # kube-score: group/version/ns/name or v1/ns/name
        rest = rest[-2:]
    if len(rest) == 2:
        return kind, rest[0] or "default", rest[1], container
    if len(rest) == 1:
        return kind, "default", rest[0], container
    return kind, "default", "unknown", container


def needs_llm(proposal: dict) -> bool:
    """The deterministic rules had no concrete patch for this issue."""
    return proposal.get("patch_type") == "yaml-snippet"


def heal(call, findings: list, manifest: str, audit, llm_fallback: bool = True) -> list:
    """Propose (never apply) remediations for findings.

    call(name, arguments, id_) -> (text, is_error). propose_remediation always
    runs first; propose_remediation_llm is only a fallback and its output is
    labelled generator=llm by the server.
    """
    proposals = []
    for i, f in enumerate(findings):
        issue = f.get("id") or f.get("message") or "unknown"
        kind, ns, name, container = parse_resource(f.get("resource") or "")
        if kind not in WORKLOAD_KINDS:
            audit("healer.skip", {"issue": issue, "resource": f.get("resource"), "reason": "not a workload"})
            continue
        base = {"kind": kind, "name": name, "namespace": ns, "issue": issue,
                "finding": f, "manifest": manifest}
        if container:
            base["container"] = container
        text, err = call("propose_remediation", base, 10 + i)
        audit("healer.propose", {"issue": issue, "resource": f.get("resource"), "is_error": err})
        if err:
            continue
        try:
            proposal = json.loads(text)
        except json.JSONDecodeError:
            proposals.append({"raw": text})
            continue
        if llm_fallback and needs_llm(proposal):
            llm_text, llm_err = call("propose_remediation_llm", base, 100 + i)
            audit("healer.propose_llm", {"issue": issue, "is_error": llm_err, "detail": llm_text[:300] if llm_err else ""})
            if not llm_err:
                try:
                    proposal = json.loads(llm_text)
                except json.JSONDecodeError:
                    pass
        proposals.append(proposal)
    return proposals


def main() -> int:
    ap = argparse.ArgumentParser(description="Reflexion agent over Starling Lite MCP")
    ap.add_argument("--mcp", default="http://127.0.0.1:8822/mcp", help="MCP HTTP endpoint")
    ap.add_argument("--path", required=True, help="Manifest path to scan")
    ap.add_argument("--trace-dir", default=None, help="Trace output dir")
    ap.add_argument(
        "--no-llm-fallback",
        dest="llm_fallback",
        action="store_false",
        help="Do not call propose_remediation_llm when the rules have no patch",
    )
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

    # Healer: deterministic propose_remediation first, guarded LLM fallback second
    findings = scan.get("findings") or []
    manifest_text = path.read_text()
    proposals = heal(
        lambda name, arguments, id_: call_tool(args.mcp, name, arguments, id_),
        findings[:5],
        manifest_text,
        audit,
        llm_fallback=args.llm_fallback,
    )

    print(f"\n=== PROPOSALS ({len(proposals)}) ===")
    for p in proposals:
        print(json.dumps(p, indent=2)[:500])
        print("---")

    audit("done", {"proposals": len(proposals), "trace": str(trace_path)})
    print(f"\ntrace written: {trace_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
