"""Unit tests for the Reflexion healer (no MCP server, no LLM)."""

import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

import run  # noqa: E402

MANIFEST = "apiVersion: apps/v1\nkind: Deployment\nmetadata: {name: api}\n"


class FakeMCP:
    def __init__(self, rules_patch_type="strategic-merge", llm_error=False):
        self.calls = []
        self.rules_patch_type = rules_patch_type
        self.llm_error = llm_error

    def __call__(self, name, arguments, id_):
        self.calls.append((name, arguments))
        if name == "propose_remediation":
            return json.dumps({"patch_type": self.rules_patch_type, "patch": "x", "dry_run": True}), False
        if name == "propose_remediation_llm":
            if self.llm_error:
                return "LLM fallback disabled: set LLM_PROVIDER", True
            return json.dumps({"patch_type": "strategic-merge", "generator": "llm", "dry_run": True,
                               "rationale": "[LLM-generated — review before applying] x"}), False
        raise AssertionError(f"unexpected tool {name}")


def noop_audit(*_a, **_k):
    pass


FINDING = {"id": "POL-linuxHardening", "resource": "Deployment/default/api/container:api"}


class ParseResourceTest(unittest.TestCase):
    def test_shapes(self):
        self.assertEqual(run.parse_resource("Deployment/default/api/container:api"),
                         ("Deployment", "default", "api", "api"))
        self.assertEqual(run.parse_resource("Deployment/apps/v1/prod/api"), ("Deployment", "prod", "api", ""))
        self.assertEqual(run.parse_resource("Pod/v1/default/dbg"), ("Pod", "default", "dbg", ""))
        self.assertEqual(run.parse_resource("Deployment/api"), ("Deployment", "default", "api", ""))
        self.assertEqual(run.parse_resource(""), ("Deployment", "default", "unknown", ""))


class HealTest(unittest.TestCase):
    def test_rules_first_no_llm(self):
        mcp = FakeMCP(rules_patch_type="strategic-merge")
        out = run.heal(mcp, [FINDING], MANIFEST, noop_audit)
        self.assertEqual([c[0] for c in mcp.calls], ["propose_remediation"])
        self.assertEqual(len(out), 1)
        args = mcp.calls[0][1]
        self.assertEqual((args["kind"], args["name"], args["namespace"], args["container"]),
                         ("Deployment", "api", "default", "api"))
        self.assertEqual(args["manifest"], MANIFEST)
        self.assertIs(args["finding"], FINDING)

    def test_llm_fallback_when_rules_have_no_patch(self):
        mcp = FakeMCP(rules_patch_type="yaml-snippet")
        out = run.heal(mcp, [FINDING], MANIFEST, noop_audit)
        self.assertEqual([c[0] for c in mcp.calls], ["propose_remediation", "propose_remediation_llm"])
        self.assertEqual(out[0]["generator"], "llm")
        self.assertTrue(out[0]["dry_run"])

    def test_llm_error_keeps_rules_hint(self):
        mcp = FakeMCP(rules_patch_type="yaml-snippet", llm_error=True)
        out = run.heal(mcp, [FINDING], MANIFEST, noop_audit)
        self.assertEqual(out[0]["patch_type"], "yaml-snippet")

    def test_fallback_disabled(self):
        mcp = FakeMCP(rules_patch_type="yaml-snippet")
        run.heal(mcp, [FINDING], MANIFEST, noop_audit, llm_fallback=False)
        self.assertEqual([c[0] for c in mcp.calls], ["propose_remediation"])

    def test_never_applies_and_skips_non_workloads(self):
        mcp = FakeMCP(rules_patch_type="yaml-snippet")
        run.heal(mcp, [FINDING, {"id": "rbac", "resource": "ClusterRole/x"}], MANIFEST, noop_audit)
        names = {c[0] for c in mcp.calls}
        self.assertNotIn("apply_remediation", names)
        self.assertEqual(sum(1 for c in mcp.calls if c[0] == "propose_remediation"), 1)


if __name__ == "__main__":
    unittest.main()
