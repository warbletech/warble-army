package llmfix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

const (
	maxPatchBytes     = 16 << 10
	maxRationaleChars = 600
)

// llmAnswer is the only accepted completion shape.
type llmAnswer struct {
	Patch     json.RawMessage `json:"patch"`
	Rationale string          `json:"rationale"`
}

// parseAnswer strictly decodes the completion: one JSON object with exactly
// "patch" (object) and "rationale" (string). Code fences are tolerated.
func parseAnswer(text string) (map[string]any, string, error) {
	s := strings.TrimSpace(text)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	if len(s) > maxCompletionLen {
		return nil, "", fmt.Errorf("completion too large")
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.DisallowUnknownFields()
	var a llmAnswer
	if err := dec.Decode(&a); err != nil {
		return nil, "", fmt.Errorf("completion is not the required JSON object {patch, rationale}: %v", err)
	}
	if dec.More() {
		return nil, "", fmt.Errorf("completion has trailing data after the JSON object")
	}
	if len(a.Patch) == 0 || len(a.Patch) > maxPatchBytes {
		return nil, "", fmt.Errorf("patch missing or larger than %d bytes", maxPatchBytes)
	}
	var patch map[string]any
	if err := json.Unmarshal(a.Patch, &patch); err != nil || patch == nil {
		return nil, "", fmt.Errorf("patch must be a JSON object (strategic-merge patch)")
	}
	return patch, sanitizeRationale(a.Rationale), nil
}

func sanitizeRationale(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > maxRationaleChars {
		s = string(r[:maxRationaleChars]) + "…"
	}
	return s
}

// target identifies the object a patch may touch.
type target struct {
	kind, apiVersion, name, namespace string
	podSpecPath                       []string
	containers                        map[string]bool
}

// keys that the LLM may never set anywhere in a patch.
var deniedKeys = map[string]string{
	"hostPath": "hostPath volumes", "hostPort": "host ports", "hostAliases": "host aliases",
	"image": "image changes", "command": "command changes", "args": "args changes",
	"env": "env changes", "envFrom": "env changes", "volumes": "volume changes",
	"volumeMounts": "volume mount changes", "serviceAccountName": "service account changes",
	"serviceAccount": "service account changes", "nodeName": "node pinning",
	"initContainers": "init containers", "ephemeralContainers": "ephemeral containers",
	"lifecycle": "lifecycle hooks", "sysctls": "sysctls", "procMount": "procMount",
	"shareProcessNamespace": "process namespace sharing", "runtimeClassName": "runtime class changes",
	"priorityClassName": "priority class changes", "tolerations": "tolerations",
}

// keys whose value must be exactly this (safe direction only).
var requiredValues = map[string]any{
	"privileged": false, "allowPrivilegeEscalation": false, "hostNetwork": false,
	"hostPID": false, "hostIPC": false, "hostUsers": false, "runAsNonRoot": true,
	"readOnlyRootFilesystem": true, "automountServiceAccountToken": false,
}

// checkDenylist enforces the structural and value denylist. It returns the
// first violation as an error.
func checkDenylist(patch map[string]any, t target) error {
	for k, v := range patch {
		switch k {
		case "apiVersion":
			if v != t.apiVersion {
				return fmt.Errorf("patch changes apiVersion (%v != %s)", v, t.apiVersion)
			}
		case "kind":
			if v != t.kind {
				return fmt.Errorf("patch changes kind (%v != %s)", v, t.kind)
			}
		case "metadata":
			m, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("metadata must be an object")
			}
			for mk, mv := range m {
				switch mk {
				case "name":
					if mv != t.name {
						return fmt.Errorf("patch changes metadata.name")
					}
				case "namespace":
					if t.namespace != "" && mv != t.namespace {
						return fmt.Errorf("patch changes metadata.namespace")
					}
				default:
					return fmt.Errorf("patch may not set metadata.%s", mk)
				}
			}
		case "spec":
			if err := checkPath(v, t, 1); err != nil {
				return err
			}
		default:
			return fmt.Errorf("patch may not set top-level field %q", k)
		}
	}
	return nil
}

// checkPath walks spec → … → pod spec; only the pod-spec path is allowed.
func checkPath(v any, t target, depth int) error {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("%s must be an object", strings.Join(t.podSpecPath[:depth], "."))
	}
	if depth == len(t.podSpecPath) {
		return checkPodSpec(m, t)
	}
	next := t.podSpecPath[depth]
	for k, sub := range m {
		if k != next {
			return fmt.Errorf("patch may only change the pod spec; %s.%s is not allowed",
				strings.Join(t.podSpecPath[:depth], "."), k)
		}
		if err := checkPath(sub, t, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func checkPodSpec(spec map[string]any, t target) error {
	if c, ok := spec["containers"]; ok {
		list, ok := c.([]any)
		if !ok {
			return fmt.Errorf("containers must be a list")
		}
		for _, it := range list {
			cm, ok := it.(map[string]any)
			if !ok {
				return fmt.Errorf("container entries must be objects")
			}
			name, _ := cm["name"].(string)
			if !t.containers[name] {
				return fmt.Errorf("patch targets unknown container %q (adding containers is not allowed)", name)
			}
		}
	}
	return walk(spec, "podSpec")
}

func walk(v any, path string) error {
	switch x := v.(type) {
	case map[string]any:
		for k, sub := range x {
			p := path + "." + k
			if strings.HasPrefix(k, "$") {
				return fmt.Errorf("patch directive %s is not allowed", p)
			}
			if why, bad := deniedKeys[k]; bad {
				return fmt.Errorf("denied field %s (%s)", p, why)
			}
			if want, ok := requiredValues[k]; ok && sub != want {
				return fmt.Errorf("denied value %s=%v (only %v allowed)", p, sub, want)
			}
			switch k {
			case "runAsUser", "runAsGroup":
				if n, ok := sub.(float64); !ok || n < 1 {
					return fmt.Errorf("denied value %s=%v (must be a non-root id > 0)", p, sub)
				}
			case "capabilities":
				cm, ok := sub.(map[string]any)
				if !ok {
					return fmt.Errorf("%s must be an object", p)
				}
				for ck := range cm {
					if ck != "drop" {
						return fmt.Errorf("denied field %s.%s (only capabilities.drop allowed)", p, ck)
					}
				}
			case "seccompProfile":
				if sm, ok := sub.(map[string]any); ok && strings.EqualFold(fmt.Sprint(sm["type"]), "Unconfined") {
					return fmt.Errorf("denied value %s.type=Unconfined", p)
				}
			}
			if err := walk(sub, p); err != nil {
				return err
			}
		}
	case []any:
		for i, sub := range x {
			if err := walk(sub, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// typedFor returns the Go type used for strategic-merge metadata.
func typedFor(kind string) any {
	switch kind {
	case "Pod":
		return &corev1.Pod{}
	case "Deployment":
		return &appsv1.Deployment{}
	case "StatefulSet":
		return &appsv1.StatefulSet{}
	case "DaemonSet":
		return &appsv1.DaemonSet{}
	case "ReplicaSet":
		return &appsv1.ReplicaSet{}
	case "ReplicationController":
		return &corev1.ReplicationController{}
	case "Job":
		return &batchv1.Job{}
	case "CronJob":
		return &batchv1.CronJob{}
	}
	return nil
}

// strictTyped checks that patch only uses real Kubernetes fields.
func strictTyped(kind string, patch map[string]any) error {
	obj := typedFor(kind)
	if obj == nil {
		return fmt.Errorf("unsupported kind %s", kind)
	}
	b, _ := json.Marshal(patch)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(obj); err != nil {
		return fmt.Errorf("patch is not a valid %s strategic-merge patch: %v", kind, err)
	}
	return nil
}

// applySMP applies patch to the original object (both JSON) using
// Kubernetes strategic-merge semantics (containers merge by name).
func applySMP(kind string, original, patch []byte) ([]byte, error) {
	obj := typedFor(kind)
	if obj == nil {
		return nil, fmt.Errorf("unsupported kind %s", kind)
	}
	return strategicpatch.StrategicMergePatch(original, patch, obj)
}

func sortedSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
