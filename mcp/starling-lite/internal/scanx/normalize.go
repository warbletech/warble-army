package scanx

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Report is a ShrikeOps report as a generic JSON object, so starling-lite works
// with both older ShrikeOps (which scored missing tools as a pass) and newer
// ones (score.complete / score.skipped_checks).
type Report = map[string]any

// toolCheck maps the ShrikeOps "tool did not run" finding IDs to what they skip.
type toolCheck struct {
	tool, category, scoreField string
}

var notRunFindings = map[string]toolCheck{
	// ShrikeOps <= 0.1.0
	"PLUTO-NOTFOUND":     {"pluto", "deprecated_api", "api_stability"},
	"POLARIS-NOTFOUND":   {"polaris", "security_policy", "security"},
	"KUBESCORE-NOTFOUND": {"kube-score", "reliability", "reliability"},
	// ShrikeOps with warbletech/shrikeops#10
	"PLUTO-NOTRUN":     {"pluto", "deprecated_api", "api_stability"},
	"POLARIS-NOTRUN":   {"polaris", "security_policy", "security"},
	"KUBESCORE-NOTRUN": {"kube-score", "reliability", "reliability"},
}

var scoreWeights = map[string]float64{"security": 0.40, "api_stability": 0.20, "reliability": 0.25, "lint_clean": 0.15}

// Normalize guarantees that a scan whose checks did not all run is reported as
// incomplete: score.complete=false, score.grade="N/A", skipped categories are
// not scored (0, excluded from a re-weighted overall), and a top-level
// "incomplete" flag + "warning" string tell the agent not to trust the grade.
func Normalize(raw []byte) (Report, error) {
	var r Report
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("decode scan report: %w", err)
	}
	score, _ := r["score"].(map[string]any)
	if score == nil {
		return r, nil
	}

	skippedTools := map[string]toolCheck{}
	if list, ok := score["skipped_checks"].([]any); ok {
		for _, it := range list {
			if m, ok := it.(map[string]any); ok {
				tool, _ := m["tool"].(string)
				for _, tc := range notRunFindings {
					if tc.tool == tool {
						skippedTools[tool] = tc
					}
				}
			}
		}
	}
	findings, _ := r["findings"].([]any)
	for _, f := range findings {
		m, _ := f.(map[string]any)
		id, _ := m["id"].(string)
		if tc, ok := notRunFindings[id]; ok {
			skippedTools[tc.tool] = tc
		}
	}
	complete, hasComplete := score["complete"].(bool)
	if len(skippedTools) == 0 && (!hasComplete || complete) {
		r["incomplete"] = false
		return r, nil
	}

	tools := make([]string, 0, len(skippedTools))
	for t := range skippedTools {
		tools = append(tools, t)
	}
	sort.Strings(tools)

	// Older ShrikeOps scored skipped categories as 100: fix them up here.
	if !hasComplete {
		var skipped []any
		notScored := map[string]bool{}
		for _, t := range tools {
			tc := skippedTools[t]
			notScored[tc.scoreField] = true
			skipped = append(skipped, map[string]any{"tool": tc.tool, "category": tc.category, "reason": "not found on PATH"})
		}
		var weighted, wsum float64
		for field, w := range scoreWeights {
			if notScored[field] {
				score[field] = 0.0
				continue
			}
			v, _ := score[field].(float64)
			weighted += v * w
			wsum += w
		}
		if wsum > 0 {
			score["overall"] = math.Round(weighted/wsum*10) / 10
		}
		score["skipped_checks"] = skipped
	}
	score["complete"] = false
	score["grade"] = "N/A"
	r["incomplete"] = true
	r["warning"] = fmt.Sprintf("INCOMPLETE SCAN: %s did not run on this server, so their categories were not scored and no grade is given. "+
		"Do not treat this result as a pass; run ShrikeOps locally with pluto, polaris and kube-score installed.", strings.Join(tools, ", "))
	return r, nil
}
