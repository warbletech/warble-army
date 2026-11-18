package scanx

// ai-* rule family: deterministic, offline checks for AI/ML workloads
// (model-weight supply chain, MCP / prompt-injection exposure, GPU pods).
// Findings use the normalized ShrikeOps shape and are scored into the
// security category with ShrikeOps severity penalties.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// AI rule ids.
const (
	RuleModelPickle       = "AI-MODEL-PICKLE"
	RuleModelUnpinned     = "AI-MODEL-UNPINNED"
	RuleTrustRemoteCode   = "AI-MODEL-TRUST-REMOTE-CODE"
	RuleMCPReadwriteOpen  = "AI-MCP-READWRITE-EXPOSED"
	RuleMCPNoAuth         = "AI-MCP-INGRESS-NOAUTH"
	RuleToolInstruction   = "AI-TOOL-INSTRUCTION-TEXT"
	RuleLethalTrifecta    = "AI-AGENT-LETHAL-TRIFECTA"
	RuleGPUNoLimit        = "AI-GPU-NO-LIMIT"
	RuleHostPathModelCach = "AI-HOSTPATH-MODEL-CACHE"
	RulePrivDevicePlugin  = "AI-PRIVILEGED-DEVICE-PLUGIN"
)

const aiDocs = "https://github.com/warbletech/warble-army/blob/main/docs/safety.md#ai-workloads-ai-rules"

// AIRule documents one rule (also used for docs/tests).
type AIRule struct {
	ID, Severity, Summary, Remediation string
}

// AIRules is the rule catalogue.
var AIRules = map[string]AIRule{
	RuleModelPickle: {RuleModelPickle, "high", "Pickle-based model weights (.bin/.pt/.pth/.pkl/.ckpt) loaded at runtime can execute code on load",
		"Use .safetensors weights; if pickle is unavoidable, scan with scan_model and load with weights_only=True."},
	RuleModelUnpinned: {RuleModelUnpinned, "medium", "Model pulled at runtime without a pinned revision SHA or digest",
		"Pin HF_REVISION/MODEL_REVISION to a 40-char commit SHA and verify MODEL_SHA256/MODEL_DIGEST before loading."},
	RuleTrustRemoteCode: {RuleTrustRemoteCode, "high", "trust_remote_code enabled: model repo Python runs in the pod",
		"Remove --trust-remote-code / TRUST_REMOTE_CODE=true, or vendor and review the model code."},
	RuleMCPReadwriteOpen: {RuleMCPReadwriteOpen, "critical", "MCP server in readwrite mode exposed through an Ingress without authentication",
		"Run the MCP server with --mode=readonly, or require auth on the Ingress (oauth2-proxy / auth-url / gateway JWT) and restrict --allow-namespaces."},
	RuleMCPNoAuth: {RuleMCPNoAuth, "medium", "MCP server exposed through an Ingress without authentication",
		"Put the MCP endpoint behind auth (auth-url / oauth2-proxy / agentgateway JWT) or keep it cluster-internal."},
	RuleToolInstruction: {RuleToolInstruction, "high", "Tool description or registry contains instruction-like text (tool poisoning / prompt injection)",
		"Keep tool descriptions declarative; remove imperative or hidden instructions and review registry changes in code review."},
	RuleLethalTrifecta: {RuleLethalTrifecta, "critical", "Agent has write/mutating tools and also ingests untrusted content (lethal trifecta)",
		"Split read-from-web and write tools into separate agents, run write tools readonly/dry-run, and require human confirmation."},
	RuleGPUNoLimit: {RuleGPUNoLimit, "high", "GPU workload without an nvidia.com/gpu limit",
		"Set resources.limits[\"nvidia.com/gpu\"] (extended resources must be set in limits; requests must equal limits)."},
	RuleHostPathModelCach: {RuleHostPathModelCach, "high", "Model cache mounted from a hostPath (shared, writable host state; cache poisoning)",
		"Use an emptyDir, a read-only PVC or an image-baked model instead of hostPath for model caches."},
	RulePrivDevicePlugin: {RulePrivDevicePlugin, "high", "Privileged GPU/device-plugin container",
		"Drop privileged: the NVIDIA device plugin only needs the kubelet device-plugins hostPath; use the GPU operator's hardened defaults."},
}

var severityPenalty = map[string]float64{"critical": 25, "high": 15, "medium": 7, "low": 3}

var (
	pickleExt     = regexp.MustCompile(`(?i)\.(bin|pt|pth|pkl|pickle|ckpt)(\?|$|["'\s])`)
	hexSHA        = regexp.MustCompile(`^[0-9a-f]{40}$`)
	modelEnvNames = map[string]bool{"HF_MODEL_ID": true, "MODEL_ID": true, "MODEL_NAME": true, "MODEL_URL": true, "MODEL_PATH": true, "MODEL_URI": true, "HF_MODEL": true}
	revEnvNames   = []string{"HF_REVISION", "MODEL_REVISION", "REVISION", "HF_MODEL_REVISION"}
	digestEnv     = []string{"MODEL_SHA256", "MODEL_DIGEST", "MODEL_CHECKSUM"}
	instructionRe = regexp.MustCompile(`(?i)(ignore (all |any )?(previous|prior|above) (instructions|rules)|disregard (the |all )?(previous|prior|above)|you must (always|now|first)|do not (tell|inform|mention) (the )?user|<\s*important\s*>|system prompt|before using this tool,? (read|send|call)|exfiltrat|send (the |all )?(contents|secrets|credentials|tokens|keys)|~/\.ssh|id_rsa|\bmcp\.json\b)`)
	writeToolRe   = regexp.MustCompile(`(?i)\b(apply_remediation|delete_pod|scale_deployment|kubectl_apply|apply_|delete_|create_|update_|patch_|write_file|exec_|run_shell|shell)\w*`)
	fetchToolRe   = regexp.MustCompile(`(?i)\b(fetch|web_fetch|http_get|browse|browser|web_search|read_url|url_fetch|scrape|crawl|github_issue|read_email|slack_read)\w*`)
)

type obj struct {
	doc       map[string]any
	kind, ns  string
	name      string
	podSpec   map[string]any
	podLabels map[string]any
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func mapOf(v any) map[string]any { m, _ := v.(map[string]any); return m }
func listOf(v any) []any         { l, _ := v.([]any); return l }

func podSpecPath(kind string) []string {
	switch kind {
	case "Pod":
		return []string{"spec"}
	case "Deployment", "StatefulSet", "DaemonSet", "ReplicaSet", "ReplicationController", "Job":
		return []string{"spec", "template", "spec"}
	case "CronJob":
		return []string{"spec", "jobTemplate", "spec", "template", "spec"}
	}
	return nil
}

func parseDocs(manifest string) []obj {
	var out []obj
	dec := yaml.NewDecoder(strings.NewReader(manifest))
	for {
		var doc map[string]any
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			break // unparseable YAML is ShrikeOps' job (lint findings)
		}
		if doc == nil {
			continue
		}
		if str(doc["kind"]) == "List" {
			for _, it := range listOf(doc["items"]) {
				if m := mapOf(it); m != nil {
					out = append(out, newObj(m))
				}
			}
			continue
		}
		out = append(out, newObj(doc))
	}
	return out
}

func newObj(doc map[string]any) obj {
	meta := mapOf(doc["metadata"])
	o := obj{doc: doc, kind: str(doc["kind"]), name: str(meta["name"]), ns: str(meta["namespace"])}
	if o.ns == "" {
		o.ns = "default"
	}
	if p := podSpecPath(o.kind); p != nil {
		var cur any = doc
		var parent map[string]any
		for _, k := range p {
			parent = mapOf(cur)
			cur = parent[k]
		}
		o.podSpec = mapOf(cur)
		if o.kind == "Pod" {
			o.podLabels = mapOf(meta["labels"])
		} else if parent != nil {
			// parent is the template; labels live at template.metadata.labels
			var t any = doc
			for _, k := range p[:len(p)-1] {
				t = mapOf(t)[k]
			}
			o.podLabels = mapOf(mapOf(mapOf(t)["metadata"])["labels"])
		}
	}
	return o
}

func (o obj) containers(withInit bool) []map[string]any {
	var out []map[string]any
	keys := []string{"containers"}
	if withInit {
		keys = append(keys, "initContainers")
	}
	for _, k := range keys {
		for _, c := range listOf(o.podSpec[k]) {
			if m := mapOf(c); m != nil {
				out = append(out, m)
			}
		}
	}
	return out
}

func envMap(c map[string]any) map[string]string {
	out := map[string]string{}
	for _, e := range listOf(c["env"]) {
		m := mapOf(e)
		if m != nil {
			out[str(m["name"])] = str(m["value"])
		}
	}
	return out
}

func cmdLine(c map[string]any) string {
	var parts []string
	for _, k := range []string{"command", "args"} {
		for _, a := range listOf(c[k]) {
			parts = append(parts, str(a))
		}
	}
	return strings.Join(parts, " ")
}

func finding(id string, o obj, container, detail string) map[string]any {
	r := AIRules[id]
	res := fmt.Sprintf("%s/%s/%s", o.kind, o.ns, o.name)
	if container != "" {
		res += "/container:" + container
	}
	msg := r.Summary
	if detail != "" {
		msg += ": " + detail
	}
	return map[string]any{
		"id": id, "category": "ai_security", "severity": r.Severity, "resource": res,
		"message": msg, "remediation": r.Remediation, "docs_url": aiDocs,
	}
}

// AIFindings runs the ai-* rules over a (multi-doc) manifest.
func AIFindings(manifest string) []map[string]any {
	objs := parseDocs(manifest)
	var out []map[string]any
	seen := map[string]bool{}
	add := func(f map[string]any) {
		k := str(f["id"]) + "|" + str(f["resource"]) + "|" + str(f["message"])
		if !seen[k] {
			seen[k] = true
			out = append(out, f)
		}
	}
	for _, o := range objs {
		if o.podSpec != nil {
			workloadRules(o, add)
		}
		if o.kind == "ConfigMap" {
			configMapRules(o, add)
		}
	}
	mcpExposureRules(objs, add)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if str(a["id"]) != str(b["id"]) {
			return str(a["id"]) < str(b["id"])
		}
		return str(a["resource"]) < str(b["resource"])
	})
	return out
}

func workloadRules(o obj, add func(map[string]any)) {
	// hostPath model caches
	hostPathVols := map[string]string{}
	for _, v := range listOf(o.podSpec["volumes"]) {
		m := mapOf(v)
		if hp := mapOf(m["hostPath"]); hp != nil {
			hostPathVols[str(m["name"])] = str(hp["path"])
		}
	}
	for _, c := range o.containers(true) {
		cname := str(c["name"])
		env := envMap(c)
		cmd := cmdLine(c)
		blob := cmd
		for k, v := range env {
			blob += " " + k + "=" + v
		}

		// --- model weights
		if loc := pickleExt.FindString(blob); loc != "" && looksLikeModelRef(blob) {
			add(finding(RuleModelPickle, o, cname, "references "+strings.Trim(loc, "\"' ?")))
		}
		modelRef := ""
		for k, v := range env {
			if modelEnvNames[k] && v != "" {
				modelRef = k + "=" + v
			}
		}
		if modelRef == "" && regexp.MustCompile(`(?i)(huggingface-cli download|hf download|--model[= ][^/.\s-]\S*|from_pretrained)`).MatchString(cmd) {
			modelRef = "command downloads a model"
		}
		if modelRef != "" {
			pinned := false
			for _, k := range revEnvNames {
				if hexSHA.MatchString(strings.ToLower(env[k])) {
					pinned = true
				}
			}
			if m := regexp.MustCompile(`--revision[= ]([0-9a-fA-F]{40})\b`).FindString(cmd); m != "" {
				pinned = true
			}
			if strings.Contains(modelRef, "@sha256:") {
				pinned = true
			}
			for _, k := range digestEnv {
				if env[k] != "" {
					pinned = true
				}
			}
			if !pinned {
				add(finding(RuleModelUnpinned, o, cname, modelRef))
			}
		}
		if strings.EqualFold(env["TRUST_REMOTE_CODE"], "true") || strings.Contains(cmd, "--trust-remote-code") || strings.Contains(cmd, "trust_remote_code=True") {
			add(finding(RuleTrustRemoteCode, o, cname, ""))
		}

		// --- hostPath model cache
		for _, vm := range listOf(c["volumeMounts"]) {
			m := mapOf(vm)
			hp, ok := hostPathVols[str(m["name"])]
			if ok && (isModelPath(str(m["mountPath"])) || isModelPath(hp)) {
				add(finding(RuleHostPathModelCach, o, cname, fmt.Sprintf("hostPath %s mounted at %s", hp, str(m["mountPath"]))))
			}
		}

		// --- GPU
		res := mapOf(c["resources"])
		_, gpuLimit := mapOf(res["limits"])["nvidia.com/gpu"]
		_, gpuReq := mapOf(res["requests"])["nvidia.com/gpu"]
		if !gpuLimit && !isInit(o, c) && (gpuReq || wantsGPU(o, c, env)) {
			add(finding(RuleGPUNoLimit, o, cname, ""))
		}
		sc := mapOf(c["securityContext"])
		if sc["privileged"] == true && isDevicePlugin(o, c) {
			add(finding(RulePrivDevicePlugin, o, cname, ""))
		}
	}
}

func isInit(o obj, c map[string]any) bool {
	for _, ic := range listOf(o.podSpec["initContainers"]) {
		if str(mapOf(ic)["name"]) == str(c["name"]) {
			return true
		}
	}
	return false
}

func looksLikeModelRef(s string) bool {
	return regexp.MustCompile(`(?i)(model|weights|checkpoint|huggingface|hf_|torch|\.pt\b|\.pth\b|\.ckpt\b|\.pkl\b)`).MatchString(s)
}

func isModelPath(p string) bool {
	return regexp.MustCompile(`(?i)(huggingface|/models?\b|model[-_]?cache|/\.cache/(huggingface|torch)|/weights|/checkpoints|ollama|vllm)`).MatchString(p)
}

func wantsGPU(o obj, c map[string]any, env map[string]string) bool {
	if _, ok := env["NVIDIA_VISIBLE_DEVICES"]; ok && env["NVIDIA_VISIBLE_DEVICES"] != "none" && env["NVIDIA_VISIBLE_DEVICES"] != "void" {
		return true
	}
	for k := range mapOf(o.podSpec["nodeSelector"]) {
		if strings.Contains(k, "nvidia.com/gpu") || strings.Contains(k, "accelerator") {
			return true
		}
	}
	for _, t := range listOf(o.podSpec["tolerations"]) {
		if strings.Contains(str(mapOf(t)["key"]), "nvidia.com/gpu") {
			return true
		}
	}
	if str(o.podSpec["runtimeClassName"]) == "nvidia" {
		return true
	}
	return false
}

func isDevicePlugin(o obj, c map[string]any) bool {
	s := strings.ToLower(o.name + " " + str(c["name"]) + " " + str(c["image"]))
	return strings.Contains(s, "device-plugin") || strings.Contains(s, "nvidia") || strings.Contains(s, "gpu-operator") || strings.Contains(s, "gpu-feature-discovery")
}

// --- ConfigMaps: tool registries / agent configs

func configMapRules(o obj, add func(map[string]any)) {
	data := mapOf(o.doc["data"])
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := str(data[k])
		descs := toolDescriptions(v)
		for _, d := range descs {
			if m := instructionRe.FindString(d); m != "" {
				f := finding(RuleToolInstruction, o, "", fmt.Sprintf("%s: %q", k, m))
				add(f)
			}
		}
		if isAgentConfig(k, v) {
			if w, f := writeToolRe.FindString(v), fetchToolRe.FindString(v); w != "" && f != "" && !strings.Contains(v, "mode: readonly") && !strings.Contains(v, `"mode": "readonly"`) {
				add(finding(RuleLethalTrifecta, o, "", fmt.Sprintf("%s: write tool %q + untrusted input %q", k, w, f)))
			}
		}
	}
}

// toolDescriptions extracts "description" strings from JSON/YAML tool registries.
func toolDescriptions(v string) []string {
	var tree any
	if err := json.Unmarshal([]byte(v), &tree); err != nil {
		if yerr := yaml.Unmarshal([]byte(v), &tree); yerr != nil {
			return nil
		}
	}
	var out []string
	var walk func(any)
	walk = func(n any) {
		switch x := n.(type) {
		case map[string]any:
			for k, sub := range x {
				if (k == "description" || k == "instructions") && str(sub) != "" {
					out = append(out, str(sub))
				}
				walk(sub)
			}
		case []any:
			for _, sub := range x {
				walk(sub)
			}
		}
	}
	walk(tree)
	return out
}

func isAgentConfig(key, v string) bool {
	k := strings.ToLower(key)
	return strings.Contains(k, "agent") || strings.Contains(k, "tools") || strings.Contains(k, "mcp") ||
		strings.Contains(v, "\"tools\"") || strings.Contains(v, "tools:")
}

// --- MCP exposure via Ingress

func isMCP(o obj) (bool, bool) {
	mcp, rw := false, false
	for _, c := range o.containers(false) {
		s := strings.ToLower(str(c["image"]) + " " + str(c["name"]) + " " + cmdLine(c))
		if strings.Contains(s, "mcp") || strings.Contains(s, "starling-lite") || strings.Contains(s, "heron-ops") || strings.Contains(s, "steadyhelm") {
			mcp = true
		}
		cmd := cmdLine(c)
		env := envMap(c)
		if regexp.MustCompile(`--mode[= ]readwrite`).MatchString(cmd) || strings.EqualFold(env["MODE"], "readwrite") ||
			strings.EqualFold(env["STARLING_MODE"], "readwrite") || strings.EqualFold(env["MCP_MODE"], "readwrite") {
			rw = true
		}
	}
	return mcp, rw
}

var authAnnotations = []string{
	"nginx.ingress.kubernetes.io/auth-url", "nginx.ingress.kubernetes.io/auth-type", "nginx.ingress.kubernetes.io/auth-signin",
	"nginx.ingress.kubernetes.io/auth-tls-secret", "traefik.ingress.kubernetes.io/router.middlewares",
	"alb.ingress.kubernetes.io/auth-type", "konghq.com/plugins", "ingress.kubernetes.io/auth-type",
	"cloud.google.com/backend-config",
}

func ingressHasAuth(ing obj) bool {
	ann := mapOf(mapOf(ing.doc["metadata"])["annotations"])
	for _, a := range authAnnotations {
		if v, ok := ann[a]; ok && str(v) != "" {
			return true
		}
	}
	return false
}

func ingressServices(ing obj) []string {
	var out []string
	spec := mapOf(ing.doc["spec"])
	addBackend := func(b map[string]any) {
		if s := mapOf(b["service"]); s != nil {
			out = append(out, str(s["name"]))
		} else if n := str(b["serviceName"]); n != "" {
			out = append(out, n)
		}
	}
	addBackend(mapOf(spec["defaultBackend"]))
	addBackend(mapOf(spec["backend"]))
	for _, r := range listOf(spec["rules"]) {
		for _, p := range listOf(mapOf(mapOf(r)["http"])["paths"]) {
			addBackend(mapOf(mapOf(p)["backend"]))
		}
	}
	return out
}

func selects(selector, labels map[string]any) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if str(labels[k]) != str(v) {
			return false
		}
	}
	return true
}

func mcpExposureRules(objs []obj, add func(map[string]any)) {
	services := map[string]obj{}
	for _, o := range objs {
		if o.kind == "Service" {
			services[o.ns+"/"+o.name] = o
		}
	}
	for _, ing := range objs {
		if ing.kind != "Ingress" || ingressHasAuth(ing) {
			continue
		}
		for _, svcName := range ingressServices(ing) {
			svc, ok := services[ing.ns+"/"+svcName]
			if !ok {
				continue
			}
			sel := mapOf(mapOf(svc.doc["spec"])["selector"])
			for _, w := range objs {
				if w.podSpec == nil || w.ns != ing.ns || !selects(sel, w.podLabels) {
					continue
				}
				mcp, rw := isMCP(w)
				if !mcp {
					continue
				}
				detail := fmt.Sprintf("Ingress %s -> Service %s -> %s/%s", ing.name, svcName, w.kind, w.name)
				if rw {
					add(finding(RuleMCPReadwriteOpen, w, "", detail))
				} else {
					add(finding(RuleMCPNoAuth, w, "", detail))
				}
			}
		}
	}
}

// --- integration with the normalized report

// MergeAI appends ai-* findings to a normalized report, updates the summary
// and folds their penalties into the security score (ShrikeOps weights and
// grade bands). Incomplete reports keep grade N/A.
func MergeAI(r Report, manifest string) Report {
	fs := AIFindings(manifest)
	if r == nil || len(fs) == 0 {
		return r
	}
	list, _ := r["findings"].([]any)
	summary := mapOf(r["summary"])
	if summary == nil {
		summary = map[string]any{}
		r["summary"] = summary
	}
	penalty := 0.0
	for _, f := range fs {
		list = append(list, f)
		sev := str(f["severity"])
		summary[sev] = num(summary[sev]) + 1
		summary["total"] = num(summary["total"]) + 1
		penalty += severityPenalty[sev]
	}
	r["findings"] = list
	r["ai_findings"] = len(fs)
	score := mapOf(r["score"])
	if score == nil {
		return r
	}
	skipped := map[string]bool{}
	for _, s := range listOf(score["skipped_checks"]) {
		for _, tc := range notRunFindings {
			if tc.tool == str(mapOf(s)["tool"]) {
				skipped[tc.scoreField] = true
			}
		}
	}
	sec := math.Max(0, num(score["security"])-penalty)
	score["security"] = math.Round(sec*10) / 10
	var weighted, wsum float64
	for field, w := range scoreWeights {
		if skipped[field] {
			continue
		}
		weighted += num(score[field]) * w
		wsum += w
	}
	if wsum > 0 {
		score["overall"] = math.Round(weighted/wsum*10) / 10
	}
	if c, ok := score["complete"].(bool); !ok || c {
		score["grade"] = gradeFor(num(score["overall"]))
	}
	return r
}

func num(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	}
	return 0
}

func gradeFor(overall float64) string {
	switch {
	case overall >= 90:
		return "A"
	case overall >= 75:
		return "B"
	case overall >= 60:
		return "C"
	case overall >= 45:
		return "D"
	}
	return "F"
}
